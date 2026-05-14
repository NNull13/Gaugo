package metric

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Canonical names for deterministic (non-judge) metrics.
const (
	NameJSONValidity     = "JSONValidity"
	NameSchemaCompliance = "SchemaCompliance"
	NameExpectedJSON     = "ExpectedJSON"
	NameAnswerSimilarity = "AnswerSimilarity"
	NameLatency          = "Latency"
	NameAnswerLength     = "AnswerLength"
	NameExpectedRegex    = "ExpectedRegex"
)

// Standard detail keys exposed on Result.Details for built-in metrics. They
// are part of the observable contract and remain stable.
const (
	DetailKeyValid              = "valid"
	DetailKeyError              = "error"
	DetailKeyFields             = "fields"
	DetailKeyElapsedMS          = "elapsed_ms"
	DetailKeyMaxMS              = "max_ms"
	DetailKeyLength             = "length"
	DetailKeyPattern            = "pattern"
	DetailKeyMatch              = "match"
	DetailKeyIntersectionTokens = "intersection_tokens"
	DetailKeyUnionTokens        = "union_tokens"
)

// JSONValidity scores whether the answer parses as valid JSON.
func JSONValidity(opts ...Option) Metric {
	cfg, err := applyOptions(opts)
	if err != nil {
		return invalidMetric{name: NameJSONValidity, err: err}
	}
	return jsonValidityMetric{cfg: cfg}
}

// SchemaCompliance scores whether the answer matches the configured JSON
// schema. Requires WithSchema.
func SchemaCompliance(opts ...Option) Metric {
	cfg, err := applyOptions(opts)
	if err != nil {
		return invalidMetric{name: NameSchemaCompliance, err: err}
	}
	if cfg.schema == nil {
		return invalidMetric{name: NameSchemaCompliance, err: optionErrorf("schema compliance metric requires WithSchema")}
	}
	return schemaComplianceMetric{cfg: cfg}
}

// ExpectedJSON scores how many expected JSON fields match the answer at the
// configured dotted paths. Requires WithExpectedFields.
func ExpectedJSON(opts ...Option) Metric {
	cfg, err := applyOptions(opts)
	if err != nil {
		return invalidMetric{name: NameExpectedJSON, err: err}
	}
	if len(cfg.expectedFields) == 0 {
		return invalidMetric{name: NameExpectedJSON, err: optionErrorf("expected json metric requires WithExpectedFields")}
	}
	return expectedJSONMetric{cfg: cfg}
}

// AnswerSimilarity scores Jaccard token overlap against Expected.Answer.
func AnswerSimilarity(opts ...Option) Metric {
	cfg, err := applyOptions(opts)
	if err != nil {
		return invalidMetric{name: NameAnswerSimilarity, err: err}
	}
	return answerSimilarityMetric{cfg: cfg}
}

// Latency scores whether the run elapsed within the configured maximum.
// Requires WithMaxLatency.
func Latency(opts ...Option) Metric {
	cfg, err := applyOptions(opts)
	if err != nil {
		return invalidMetric{name: NameLatency, err: err}
	}
	if cfg.maxLatency == nil {
		return invalidMetric{name: NameLatency, err: optionErrorf("latency metric requires WithMaxLatency")}
	}
	return latencyMetric{cfg: cfg}
}

// AnswerLength scores whether the answer length lies in [min,max] runes.
// Requires WithMinLength or WithMaxLength.
func AnswerLength(opts ...Option) Metric {
	cfg, err := applyOptions(opts)
	if err != nil {
		return invalidMetric{name: NameAnswerLength, err: err}
	}
	if cfg.minLength == nil && cfg.maxLength == nil {
		return invalidMetric{name: NameAnswerLength, err: optionErrorf("answer length metric requires WithMinLength or WithMaxLength")}
	}
	return answerLengthMetric{cfg: cfg}
}

// ExpectedRegex scores whether the answer matches the given Go regular
// expression.
func ExpectedRegex(pattern string, opts ...Option) Metric {
	if strings.TrimSpace(pattern) == "" {
		return invalidMetric{name: NameExpectedRegex, err: optionErrorf("expected regex pattern must be non-empty")}
	}
	cfg, err := applyOptions(opts)
	if err != nil {
		return invalidMetric{name: NameExpectedRegex, err: err}
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return invalidMetric{name: NameExpectedRegex, err: wrapError(kindMetric, "expected regex pattern invalid", err)}
	}
	return expectedRegexMetric{pattern: pattern, re: re, cfg: cfg}
}

type jsonValidityMetric struct {
	cfg config
}

func (m jsonValidityMetric) Name() string { return NameJSONValidity }

func (m jsonValidityMetric) Evaluate(_ context.Context, in EvalInput, _ Judge) (Result, error) {
	valid := json.Valid([]byte(in.Output.Answer))
	reason := "valid JSON"
	if !valid {
		reason = "answer is not valid JSON"
	}
	return deterministicResult(m.Name(), boolScore(valid), m.cfg.threshold, reason, map[string]any{
		DetailKeyValid: valid,
	})
}

type schemaComplianceMetric struct {
	cfg config
}

func (m schemaComplianceMetric) Name() string { return NameSchemaCompliance }

func (m schemaComplianceMetric) Evaluate(_ context.Context, in EvalInput, _ Judge) (Result, error) {
	value, err := decodeJSONValue(in.Output.Answer, true)
	if err != nil {
		return deterministicResult(m.Name(), 0, m.cfg.threshold, "answer is not valid JSON", map[string]any{
			DetailKeyValid: false,
			DetailKeyError: err.Error(),
		})
	}
	if err := validateSchemaValue(value, *m.cfg.schema, "$"); err != nil {
		return deterministicResult(m.Name(), 0, m.cfg.threshold, err.Error(), map[string]any{
			DetailKeyValid: false,
			DetailKeyError: err.Error(),
		})
	}
	return deterministicResult(m.Name(), 1, m.cfg.threshold, "answer matches schema", map[string]any{
		DetailKeyValid: true,
	})
}

type expectedJSONMetric struct {
	cfg config
}

func (m expectedJSONMetric) Name() string { return NameExpectedJSON }

type expectedJSONExpectation struct {
	Path     string
	Expected any
}

type expectedJSONFieldResult struct {
	Path     string `json:"path"`
	Found    bool   `json:"found"`
	Match    bool   `json:"match"`
	Expected any    `json:"expected"`
	Actual   any    `json:"actual,omitempty"`
}

func (m expectedJSONMetric) Evaluate(_ context.Context, in EvalInput, _ Judge) (Result, error) {
	value, err := decodeJSONValue(in.Output.Answer, true)
	if err != nil {
		return deterministicResult(m.Name(), 0, m.cfg.threshold, "answer is not valid JSON", map[string]any{
			DetailKeyValid: false,
			DetailKeyError: err.Error(),
		})
	}

	results := make([]expectedJSONFieldResult, 0, len(m.cfg.expectedFields))
	matched := 0
	for _, field := range m.cfg.expectedFields {
		actual, found := lookupJSONPath(value, field.Path)
		match := found && jsonValuesEqual(actual, field.Expected)
		if match {
			matched++
		}
		results = append(results, expectedJSONFieldResult{
			Path:     field.Path,
			Found:    found,
			Match:    match,
			Expected: field.Expected,
			Actual:   actual,
		})
	}

	score := float64(matched) / float64(len(m.cfg.expectedFields))
	reason := fmt.Sprintf("%d/%d expected JSON fields matched", matched, len(m.cfg.expectedFields))
	return deterministicResult(m.Name(), score, m.cfg.threshold, reason, map[string]any{
		DetailKeyFields: results,
	})
}

type answerSimilarityMetric struct {
	cfg config
}

func (m answerSimilarityMetric) Name() string { return NameAnswerSimilarity }

func (m answerSimilarityMetric) Evaluate(_ context.Context, in EvalInput, _ Judge) (Result, error) {
	expected := strings.TrimSpace(in.Expected.Answer)
	if expected == "" {
		return Result{}, requiresExpectedAnswerError(labelOf(NameAnswerSimilarity))
	}
	score, intersection, union := jaccardSimilarity(in.Output.Answer, expected)
	reason := fmt.Sprintf("jaccard similarity %.3f over %d shared tokens and %d total tokens", score, intersection, union)
	return deterministicResult(m.Name(), score, m.cfg.threshold, reason, map[string]any{
		DetailKeyIntersectionTokens: intersection,
		DetailKeyUnionTokens:        union,
	})
}

type latencyMetric struct {
	cfg config
}

func (m latencyMetric) Name() string { return NameLatency }

func (m latencyMetric) Evaluate(_ context.Context, in EvalInput, _ Judge) (Result, error) {
	max := *m.cfg.maxLatency
	pass := in.Elapsed <= max
	reason := fmt.Sprintf("elapsed %s <= max %s", in.Elapsed, max)
	if !pass {
		reason = fmt.Sprintf("elapsed %s > max %s", in.Elapsed, max)
	}
	return deterministicResult(m.Name(), boolScore(pass), m.cfg.threshold, reason, map[string]any{
		DetailKeyElapsedMS: float64(in.Elapsed) / float64(time.Millisecond),
		DetailKeyMaxMS:     float64(max) / float64(time.Millisecond),
	})
}

type answerLengthMetric struct {
	cfg config
}

func (m answerLengthMetric) Name() string { return NameAnswerLength }

func (m answerLengthMetric) Evaluate(_ context.Context, in EvalInput, _ Judge) (Result, error) {
	length := utf8.RuneCountInString(in.Output.Answer)
	pass := true
	bounds := make([]string, 0, 2)
	if m.cfg.minLength != nil {
		if length < *m.cfg.minLength {
			pass = false
		}
		bounds = append(bounds, fmt.Sprintf("min=%d", *m.cfg.minLength))
	}
	if m.cfg.maxLength != nil {
		if length > *m.cfg.maxLength {
			pass = false
		}
		bounds = append(bounds, fmt.Sprintf("max=%d", *m.cfg.maxLength))
	}
	format := "answer length %d characters within %s"
	if !pass {
		format = "answer length %d characters outside %s"
	}
	reason := fmt.Sprintf(format, length, strings.Join(bounds, " "))
	return deterministicResult(m.Name(), boolScore(pass), m.cfg.threshold, reason, map[string]any{
		DetailKeyLength: length,
	})
}

type expectedRegexMetric struct {
	pattern string
	re      *regexp.Regexp
	cfg     config
}

func (m expectedRegexMetric) Name() string { return NameExpectedRegex }

func (m expectedRegexMetric) Evaluate(_ context.Context, in EvalInput, _ Judge) (Result, error) {
	match := m.re.MatchString(in.Output.Answer)
	reason := "answer matched expected regex"
	if !match {
		reason = "answer did not match expected regex"
	}
	return deterministicResult(m.Name(), boolScore(match), m.cfg.threshold, reason, map[string]any{
		DetailKeyPattern: m.pattern,
		DetailKeyMatch:   match,
	})
}
