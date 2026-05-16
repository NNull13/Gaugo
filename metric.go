package gaugo

import (
	"encoding/json"
	"time"

	"github.com/nnull13/gaugo/metric"
)

type (
	Metric        = metric.Metric
	MetricResult  = metric.Result
	MetricOption  = metric.Option
	Judge         = metric.Judge
	JudgeRequest  = metric.JudgeRequest
	JudgeResponse = metric.JudgeResponse
	EvalInput     = metric.EvalInput
	Input         = metric.Input
	Output        = metric.Output
	Expected      = metric.Expected
	Document      = metric.Document
	FuncJudge     = metric.FuncJudge
)

// Doc is a convenience constructor for Document.
func Doc(id, text string) Document { return metric.Doc(id, text) }

// WithThreshold sets the pass/fail threshold in [0,1]. Default is 0.7.
func WithThreshold(v float64) MetricOption { return metric.WithThreshold(v) }

// WithSchema sets the JSON Schema used by SchemaCompliance.
func WithSchema(schema json.RawMessage) MetricOption { return metric.WithSchema(schema) }

// WithExpectedFields sets the dotted JSON paths and expected values used by ExpectedJSON.
func WithExpectedFields(fields map[string]any) MetricOption { return metric.WithExpectedFields(fields) }

// WithMaxLatency sets the maximum allowed run latency for Latency.
func WithMaxLatency(d time.Duration) MetricOption { return metric.WithMaxLatency(d) }

// WithMinLength sets the minimum allowed answer length in runes.
func WithMinLength(n int) MetricOption { return metric.WithMinLength(n) }

// WithMaxLength sets the maximum allowed answer length in runes.
func WithMaxLength(n int) MetricOption { return metric.WithMaxLength(n) }

// Judge-based metrics.

// Faithfulness scores whether the answer is supported by the provided context.
func Faithfulness(opts ...MetricOption) Metric { return metric.Faithfulness(opts...) }

// AnswerRelevancy scores how well the answer addresses the question.
func AnswerRelevancy(opts ...MetricOption) Metric { return metric.AnswerRelevancy(opts...) }

// ContextRelevancy scores how relevant each context document is to the question.
func ContextRelevancy(opts ...MetricOption) Metric { return metric.ContextRelevancy(opts...) }

// ContextPrecision scores the fraction of context documents that are useful.
func ContextPrecision(opts ...MetricOption) Metric { return metric.ContextPrecision(opts...) }

// ContextRecall scores how much of the expected answer is supported by the context.
func ContextRecall(opts ...MetricOption) Metric { return metric.ContextRecall(opts...) }

// AnswerCorrectness scores how well the answer matches Expected.Answer.
func AnswerCorrectness(opts ...MetricOption) Metric { return metric.AnswerCorrectness(opts...) }

// Hallucination scores the fraction of claims that are not hallucinated.
func Hallucination(opts ...MetricOption) Metric { return metric.Hallucination(opts...) }

// Toxicity scores how non-toxic the answer is.
func Toxicity(opts ...MetricOption) Metric { return metric.Toxicity(opts...) }

// Bias scores how unbiased the answer is.
func Bias(opts ...MetricOption) Metric { return metric.Bias(opts...) }

// Coherence scores how internally consistent the answer is.
func Coherence(opts ...MetricOption) Metric { return metric.Coherence(opts...) }

// Conciseness scores how succinct the answer is without losing meaning.
func Conciseness(opts ...MetricOption) Metric { return metric.Conciseness(opts...) }

// Completeness scores how completely the answer addresses the question.
func Completeness(opts ...MetricOption) Metric { return metric.Completeness(opts...) }

// InstructionAdherence scores how closely the answer followed Expected.Instructions.
func InstructionAdherence(opts ...MetricOption) Metric { return metric.InstructionAdherence(opts...) }

// CitationAccuracy scores how accurate inline citations are against context.
func CitationAccuracy(opts ...MetricOption) Metric { return metric.CitationAccuracy(opts...) }

// SummarizationQuality scores summary coverage, fidelity and conciseness.
func SummarizationQuality(opts ...MetricOption) Metric { return metric.SummarizationQuality(opts...) }

// GEval scores along a free-form criteria string. Criteria must be non-empty.
func GEval(criteria string, opts ...MetricOption) Metric { return metric.GEval(criteria, opts...) }

// Deterministic metrics.

// JSONValidity scores whether the answer parses as valid JSON.
func JSONValidity(opts ...MetricOption) Metric { return metric.JSONValidity(opts...) }

// SchemaCompliance scores whether the answer matches the configured JSON schema.
func SchemaCompliance(opts ...MetricOption) Metric { return metric.SchemaCompliance(opts...) }

// ExpectedJSON scores how many expected JSON fields match the answer.
func ExpectedJSON(opts ...MetricOption) Metric { return metric.ExpectedJSON(opts...) }

// AnswerSimilarity scores Jaccard token overlap against Expected.Answer.
func AnswerSimilarity(opts ...MetricOption) Metric { return metric.AnswerSimilarity(opts...) }

// Latency scores whether the run elapsed within the configured maximum.
func Latency(opts ...MetricOption) Metric { return metric.Latency(opts...) }

// AnswerLength scores whether the answer length lies in [min,max] runes.
func AnswerLength(opts ...MetricOption) Metric { return metric.AnswerLength(opts...) }

// ExpectedRegex scores whether the answer matches a Go regular expression.
func ExpectedRegex(pattern string, opts ...MetricOption) Metric {
	return metric.ExpectedRegex(pattern, opts...)
}
