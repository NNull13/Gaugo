package gaugo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nnull13/gaugo/internal/metrics/answerrelevancy"
	"github.com/nnull13/gaugo/internal/metrics/contextrelevancy"
	"github.com/nnull13/gaugo/internal/metrics/faithfulness"
	"github.com/nnull13/gaugo/internal/prompt"
)

const (
	metricNameFaithfulness     = "Faithfulness"
	metricNameAnswerRelevancy  = "AnswerRelevancy"
	metricNameContextRelevancy = "ContextRelevancy"

	errJudgeEvaluationFailed = "judge evaluation failed: %w"
	errMetricRequiresJudge   = "%s metric requires a configured judge"
	errMetricParseFailed     = "%s parse failed: %w"
	errMetricMarshalFailed   = "%s marshal details failed: %w"

	metricLabelFaithfulness     = "faithfulness"
	metricLabelAnswerRelevancy  = "answer relevancy"
	metricLabelContextRelevancy = "context relevancy"
)

// Metric evaluates a completed case and returns a score plus pass/fail result.
type Metric interface {
	Name() string
	Evaluate(ctx context.Context, in EvalInput, j Judge) (MetricResult, error)
}

type metricConfig struct {
	threshold float64
	err       error
}

// MetricOption configures a built-in metric.
type MetricOption func(*metricConfig)

// WithThreshold sets pass/fail threshold in [0,1].
func WithThreshold(v float64) MetricOption {
	return func(c *metricConfig) {
		if v < 0 || v > 1 {
			c.err = fmt.Errorf("threshold must be in [0,1], got %f", v)
			return
		}
		c.threshold = v
	}
}

func applyMetricOptions(opts []MetricOption) (metricConfig, error) {
	cfg := metricConfig{threshold: 0.7}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.err != nil {
		return metricConfig{}, cfg.err
	}
	return cfg, nil
}

// Faithfulness returns a metric that scores whether the answer is supported by context.
func Faithfulness(opts ...MetricOption) Metric {
	cfg, err := applyMetricOptions(opts)
	if err != nil {
		return invalidMetric{name: metricNameFaithfulness, err: err}
	}
	return faithfulnessMetric{cfg: cfg}
}

// AnswerRelevancy returns a metric that scores whether the answer addresses the input.
func AnswerRelevancy(opts ...MetricOption) Metric {
	cfg, err := applyMetricOptions(opts)
	if err != nil {
		return invalidMetric{name: metricNameAnswerRelevancy, err: err}
	}
	return answerRelevancyMetric{cfg: cfg}
}

// ContextRelevancy returns a metric that scores whether context documents are relevant to the input.
func ContextRelevancy(opts ...MetricOption) Metric {
	cfg, err := applyMetricOptions(opts)
	if err != nil {
		return invalidMetric{name: metricNameContextRelevancy, err: err}
	}
	return contextRelevancyMetric{cfg: cfg}
}

type invalidMetric struct {
	name string
	err  error
}

func (m invalidMetric) Name() string { return m.name }

func (m invalidMetric) Evaluate(context.Context, EvalInput, Judge) (MetricResult, error) {
	return MetricResult{}, m.err
}

type faithfulnessMetric struct {
	cfg metricConfig
}

func (m faithfulnessMetric) Name() string { return metricNameFaithfulness }

func (m faithfulnessMetric) Evaluate(ctx context.Context, in EvalInput, j Judge) (MetricResult, error) {
	if j == nil {
		return MetricResult{}, fmt.Errorf(errMetricRequiresJudge, metricLabelFaithfulness)
	}

	resp, err := j.EvaluateJSON(ctx, JudgeRequest{
		Metric:       m.Name(),
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.FaithfulnessInstructions(),
		Schema:       prompt.FaithfulnessSchema(),
	})
	if err != nil {
		return MetricResult{}, fmt.Errorf(errJudgeEvaluationFailed, err)
	}

	var parsed faithfulness.Output
	parsed, err = faithfulness.Parse(resp.RawJSON)
	if err != nil {
		return MetricResult{}, fmt.Errorf(errMetricParseFailed, metricLabelFaithfulness, err)
	}

	score := faithfulness.Score(parsed.Claims)

	var details []byte
	details, err = json.Marshal(parsed)
	if err != nil {
		return MetricResult{}, fmt.Errorf(errMetricMarshalFailed, metricLabelFaithfulness, err)
	}

	return MetricResult{
		Name:    m.Name(),
		Score:   score,
		Pass:    score >= m.cfg.threshold,
		Reason:  parsed.Reason,
		Details: details,
	}, nil
}

type contextRelevancyMetric struct {
	cfg metricConfig
}

func (m contextRelevancyMetric) Name() string { return metricNameContextRelevancy }

func (m contextRelevancyMetric) Evaluate(ctx context.Context, in EvalInput, j Judge) (MetricResult, error) {
	if j == nil {
		return MetricResult{}, fmt.Errorf(errMetricRequiresJudge, metricLabelContextRelevancy)
	}

	resp, err := j.EvaluateJSON(ctx, JudgeRequest{
		Metric:       m.Name(),
		Question:     in.Input.Question,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.ContextRelevancyInstructions(),
		Schema:       prompt.ContextRelevancySchema(),
	})
	if err != nil {
		return MetricResult{}, fmt.Errorf(errJudgeEvaluationFailed, err)
	}

	var parsed contextrelevancy.Output
	parsed, err = contextrelevancy.Parse(resp.RawJSON)
	if err != nil {
		return MetricResult{}, fmt.Errorf(errMetricParseFailed, metricLabelContextRelevancy, err)
	}

	score := contextrelevancy.Score(parsed.Documents)

	var details []byte
	details, err = json.Marshal(parsed)
	if err != nil {
		return MetricResult{}, fmt.Errorf(errMetricMarshalFailed, metricLabelContextRelevancy, err)
	}

	return MetricResult{
		Name:    m.Name(),
		Score:   score,
		Pass:    score >= m.cfg.threshold,
		Reason:  parsed.Reason,
		Details: details,
	}, nil
}

type answerRelevancyMetric struct {
	cfg metricConfig
}

func (m answerRelevancyMetric) Name() string { return metricNameAnswerRelevancy }

func (m answerRelevancyMetric) Evaluate(ctx context.Context, in EvalInput, j Judge) (MetricResult, error) {
	if j == nil {
		return MetricResult{}, fmt.Errorf(errMetricRequiresJudge, metricLabelAnswerRelevancy)
	}

	resp, err := j.EvaluateJSON(ctx, JudgeRequest{
		Metric:       m.Name(),
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.AnswerRelevancyInstructions(),
		Schema:       prompt.AnswerRelevancySchema(),
	})
	if err != nil {
		return MetricResult{}, fmt.Errorf(errJudgeEvaluationFailed, err)
	}

	var parsed answerrelevancy.Output
	parsed, err = answerrelevancy.Parse(resp.RawJSON)
	if err != nil {
		return MetricResult{}, fmt.Errorf(errMetricParseFailed, metricLabelAnswerRelevancy, err)
	}

	var details []byte
	details, err = json.Marshal(parsed)
	if err != nil {
		return MetricResult{}, fmt.Errorf(errMetricMarshalFailed, metricLabelAnswerRelevancy, err)
	}

	return MetricResult{
		Name:    m.Name(),
		Score:   parsed.Score,
		Pass:    parsed.Score >= m.cfg.threshold,
		Reason:  parsed.Reason,
		Details: details,
	}, nil
}
