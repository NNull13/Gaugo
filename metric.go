package gaugo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nnull13/gaugo/internal/metrics/answerrelevancy"
	"github.com/nnull13/gaugo/internal/metrics/faithfulness"
	"github.com/nnull13/gaugo/internal/prompt"
)

const (
	metricNameFaithfulness    = "Faithfulness"
	metricNameAnswerRelevancy = "AnswerRelevancy"
	judgeEvaluationFailed     = "judge evaluation failed: %w"
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

func metricDefaults() metricConfig {
	return metricConfig{threshold: 0.7}
}

// Faithfulness returns a metric that scores whether the answer is supported by context.
func Faithfulness(opts ...MetricOption) Metric {
	cfg := metricDefaults()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&cfg)
	}
	if cfg.err != nil {
		return invalidMetric{name: metricNameFaithfulness, err: cfg.err}
	}
	return faithfulnessMetric{cfg: cfg}
}

// AnswerRelevancy returns a metric that scores whether the answer addresses the input.
func AnswerRelevancy(opts ...MetricOption) Metric {
	cfg := metricDefaults()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&cfg)
	}
	if cfg.err != nil {
		return invalidMetric{name: metricNameAnswerRelevancy, err: cfg.err}
	}
	return answerRelevancyMetric{cfg: cfg}
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
		return MetricResult{}, fmt.Errorf("faithfulness metric requires a configured judge")
	}

	var err error

	var (
		resp    JudgeResponse
		parsed  faithfulness.Output
		details []byte
	)

	resp, err = j.EvaluateJSON(ctx, JudgeRequest{
		Metric:       m.Name(),
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.FaithfulnessInstructions(),
		Schema:       prompt.FaithfulnessSchema(),
	})
	if err != nil {
		return MetricResult{}, fmt.Errorf(judgeEvaluationFailed, err)
	}

	parsed, err = faithfulness.Parse(resp.RawJSON)
	if err != nil {
		return MetricResult{}, fmt.Errorf("faithfulness parse failed: %w", err)
	}

	score := faithfulness.Score(parsed.Claims)
	pass := score >= m.cfg.threshold
	details, err = json.Marshal(parsed)
	if err != nil {
		return MetricResult{}, fmt.Errorf("faithfulness marshal details failed: %w", err)
	}

	return MetricResult{
		Name:    m.Name(),
		Score:   score,
		Pass:    pass,
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
		return MetricResult{}, fmt.Errorf("answer relevancy metric requires a configured judge")
	}

	var err error

	var (
		resp    JudgeResponse
		parsed  answerrelevancy.Output
		details []byte
	)

	resp, err = j.EvaluateJSON(ctx, JudgeRequest{
		Metric:       m.Name(),
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.AnswerRelevancyInstructions(),
		Schema:       prompt.AnswerRelevancySchema(),
	})
	if err != nil {
		return MetricResult{}, fmt.Errorf(judgeEvaluationFailed, err)
	}

	parsed, err = answerrelevancy.Parse(resp.RawJSON)
	if err != nil {
		return MetricResult{}, fmt.Errorf("answer relevancy parse failed: %w", err)
	}

	details, err = json.Marshal(parsed)
	if err != nil {
		return MetricResult{}, fmt.Errorf("answer relevancy marshal details failed: %w", err)
	}

	return MetricResult{
		Name:    m.Name(),
		Score:   parsed.Score,
		Pass:    parsed.Score >= m.cfg.threshold,
		Reason:  parsed.Reason,
		Details: details,
	}, nil
}
