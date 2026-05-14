// Package metric defines the Metric interface used by gaugo to evaluate one
// case at a time, plus the built-in deterministic and LLM-judge metrics.
//
// Users may implement Metric directly to plug custom evaluation logic into
// gaugo.Suite.Assert / gaugo.Runner.Run. The root gaugo package re-exports
// every public identifier in this package as a type alias or one-line
// wrapper, so callers can use either gaugo.Faithfulness or
// metric.Faithfulness interchangeably.
package metric

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"
)

const defaultThreshold = 0.7

// Result represents a score and pass/fail outcome for one metric. Re-exported
// as gaugo.MetricResult.
type Result struct {
	Name         string
	Score        float64
	Pass         bool
	Reason       string
	Details      []byte
	Provider     string
	Model        string
	RequestID    string
	JudgeLatency time.Duration
}

// Metric evaluates a completed case and returns a Result. Built-in metrics
// implement this interface; users may also implement it for custom logic.
type Metric interface {
	Name() string
	Evaluate(ctx context.Context, in EvalInput, j Judge) (Result, error)
}

// Option configures a built-in metric. Options are validated eagerly; invalid
// combinations surface as a metric-level error during Evaluate.
type Option func(*config)

// config collects the optional knobs shared by all built-in metrics. Each
// metric only consults the fields it needs.
type config struct {
	threshold      float64
	schema         *json.RawMessage
	expectedFields []expectedJSONExpectation
	maxLatency     *time.Duration
	minLength      *int
	maxLength      *int
	err            error
}

// WithThreshold sets the pass/fail threshold in [0,1]. Default is 0.7.
func WithThreshold(v float64) Option {
	return func(c *config) {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
			c.err = optionErrorf("threshold must be in [0,1], got %f", v)
			return
		}
		c.threshold = v
	}
}

// WithSchema sets the JSON Schema used by SchemaCompliance.
func WithSchema(schema json.RawMessage) Option {
	return func(c *config) {
		if len(bytes.TrimSpace(schema)) == 0 {
			c.err = optionErrorf("schema must be non-empty")
			return
		}
		if !json.Valid(schema) {
			c.err = optionErrorf("schema must be valid JSON")
			return
		}
		copied := append(json.RawMessage(nil), schema...)
		c.schema = &copied
	}
}

// WithExpectedFields sets the dotted JSON paths and expected values used by
// ExpectedJSON. Keys are trimmed; an empty key is rejected.
func WithExpectedFields(fields map[string]any) Option {
	return func(c *config) {
		if len(fields) == 0 {
			c.err = optionErrorf("expected fields must be non-empty")
			return
		}
		byPath := make(map[string]any, len(fields))
		for k, v := range fields {
			key := strings.TrimSpace(k)
			if key == "" {
				c.err = optionErrorf("expected field paths must be non-empty")
				return
			}
			if _, exists := byPath[key]; exists {
				c.err = optionErrorf("expected field path %q is duplicated after trimming", key)
				return
			}
			normalized, err := normalizeJSONComparable(v)
			if err != nil {
				c.err = optionErrorf("expected field %q must be JSON-compatible: %v", key, err)
				return
			}
			byPath[key] = normalized
		}
		paths := make([]string, 0, len(byPath))
		for path := range byPath {
			paths = append(paths, path)
		}
		sort.Strings(paths)

		expected := make([]expectedJSONExpectation, 0, len(paths))
		for _, path := range paths {
			expected = append(expected, expectedJSONExpectation{
				Path:     path,
				Expected: byPath[path],
			})
		}
		c.expectedFields = expected
	}
}

// WithMaxLatency sets the maximum allowed run latency for Latency.
func WithMaxLatency(d time.Duration) Option {
	return func(c *config) {
		if d < 0 {
			c.err = optionErrorf("max latency must be non-negative")
			return
		}
		c.maxLatency = &d
	}
}

// WithMinLength sets the minimum allowed answer length in runes.
func WithMinLength(n int) Option {
	return func(c *config) {
		if n < 0 {
			c.err = optionErrorf("min length must be non-negative")
			return
		}
		c.minLength = &n
	}
}

// WithMaxLength sets the maximum allowed answer length in runes.
func WithMaxLength(n int) Option {
	return func(c *config) {
		if n < 0 {
			c.err = optionErrorf("max length must be non-negative")
			return
		}
		c.maxLength = &n
	}
}

func applyOptions(opts []Option) (config, error) {
	cfg := config{threshold: defaultThreshold}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&cfg)
	}
	if cfg.err != nil {
		return config{}, cfg.err
	}
	if cfg.minLength != nil && cfg.maxLength != nil && *cfg.minLength > *cfg.maxLength {
		return config{}, optionErrorf("min length must be <= max length")
	}
	return cfg, nil
}

// invalidMetric carries a deferred construction error. Returning it from a
// constructor lets us surface the error at Evaluate time without changing the
// constructor's return signature.
type invalidMetric struct {
	name string
	err  error
}

func (m invalidMetric) Name() string { return m.name }

func (m invalidMetric) Evaluate(context.Context, EvalInput, Judge) (Result, error) {
	return Result{}, m.err
}
