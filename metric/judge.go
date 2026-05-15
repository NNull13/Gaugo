package metric

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/nnull13/gaugo/internal/metrics/answercorrectness"
	"github.com/nnull13/gaugo/internal/metrics/answerrelevancy"
	"github.com/nnull13/gaugo/internal/metrics/bias"
	"github.com/nnull13/gaugo/internal/metrics/citationaccuracy"
	"github.com/nnull13/gaugo/internal/metrics/coherence"
	"github.com/nnull13/gaugo/internal/metrics/completeness"
	"github.com/nnull13/gaugo/internal/metrics/conciseness"
	"github.com/nnull13/gaugo/internal/metrics/contextprecision"
	"github.com/nnull13/gaugo/internal/metrics/contextrecall"
	"github.com/nnull13/gaugo/internal/metrics/contextrelevancy"
	"github.com/nnull13/gaugo/internal/metrics/faithfulness"
	"github.com/nnull13/gaugo/internal/metrics/hallucination"
	"github.com/nnull13/gaugo/internal/metrics/instructionadherence"
	"github.com/nnull13/gaugo/internal/metrics/summarizationquality"
	"github.com/nnull13/gaugo/internal/metrics/toxicity"
	"github.com/nnull13/gaugo/internal/prompt"
)

// Canonical metric names (the strings returned by Metric.Name and stored in
// Result.Name). Each metric's human-readable label is derived from its name
// via labelOf at runtime.
const (
	NameFaithfulness         = "Faithfulness"
	NameAnswerRelevancy      = "AnswerRelevancy"
	NameContextRelevancy     = "ContextRelevancy"
	NameContextPrecision     = "ContextPrecision"
	NameContextRecall        = "ContextRecall"
	NameAnswerCorrectness    = "AnswerCorrectness"
	NameHallucination        = "Hallucination"
	NameToxicity             = "Toxicity"
	NameBias                 = "Bias"
	NameCoherence            = "Coherence"
	NameConciseness          = "Conciseness"
	NameCompleteness         = "Completeness"
	NameInstructionAdherence = "InstructionAdherence"
	NameGEval                = "GEval"
	NameCitationAccuracy     = "CitationAccuracy"
	NameSummarizationQuality = "SummarizationQuality"
)

// Faithfulness scores whether the answer is supported by the provided context.
func Faithfulness(opts ...Option) Metric {
	return newJudgeMetric(opts, NameFaithfulness,
		faithfulnessRequest, faithfulness.Parse,
		func(o faithfulness.Output) float64 { return faithfulness.Score(o.Claims) },
		func(o faithfulness.Output) string { return o.Reason },
	)
}

// AnswerRelevancy scores how well the answer addresses the question.
func AnswerRelevancy(opts ...Option) Metric {
	return newJudgeMetric(opts, NameAnswerRelevancy,
		answerRelevancyRequest, answerrelevancy.Parse,
		func(o answerrelevancy.Output) float64 { return o.Score },
		func(o answerrelevancy.Output) string { return o.Reason },
	)
}

// ContextRelevancy scores how relevant each context document is to the question.
func ContextRelevancy(opts ...Option) Metric {
	return newJudgeMetric(opts, NameContextRelevancy,
		contextRelevancyRequest, contextrelevancy.Parse,
		func(o contextrelevancy.Output) float64 { return contextrelevancy.Score(o.Documents) },
		func(o contextrelevancy.Output) string { return o.Reason },
	)
}

// ContextPrecision scores the fraction of context documents that are useful.
func ContextPrecision(opts ...Option) Metric {
	return newJudgeMetric(opts, NameContextPrecision,
		contextPrecisionRequest, contextprecision.Parse,
		func(o contextprecision.Output) float64 { return contextprecision.Score(o.Documents) },
		func(o contextprecision.Output) string { return o.Reason },
	)
}

// ContextRecall scores how much of the expected answer is supported by the
// context. Requires Expected.Answer.
func ContextRecall(opts ...Option) Metric {
	return newJudgeMetric(opts, NameContextRecall,
		contextRecallRequest, contextrecall.Parse,
		func(o contextrecall.Output) float64 { return contextrecall.Score(o.Claims) },
		func(o contextrecall.Output) string { return o.Reason },
	)
}

// AnswerCorrectness scores how well the answer matches Expected.Answer.
// Requires Expected.Answer.
func AnswerCorrectness(opts ...Option) Metric {
	return newJudgeMetric(opts, NameAnswerCorrectness,
		answerCorrectnessRequest, answercorrectness.Parse,
		answercorrectness.Score,
		func(o answercorrectness.Output) string { return o.Reason },
	)
}

// Hallucination scores the fraction of claims that are not hallucinated.
func Hallucination(opts ...Option) Metric {
	return newJudgeMetric(opts, NameHallucination,
		hallucinationRequest, hallucination.Parse,
		func(o hallucination.Output) float64 { return hallucination.Score(o.Claims) },
		func(o hallucination.Output) string { return o.Reason },
	)
}

// Toxicity scores how non-toxic the answer is (higher is safer).
func Toxicity(opts ...Option) Metric {
	return newJudgeMetric(opts, NameToxicity,
		toxicityRequest, toxicity.Parse,
		toxicity.Score,
		func(o toxicity.Output) string { return o.Reason },
	)
}

// Bias scores how unbiased the answer is (higher is fairer).
func Bias(opts ...Option) Metric {
	return newJudgeMetric(opts, NameBias,
		biasRequest, bias.Parse,
		bias.Score,
		func(o bias.Output) string { return o.Reason },
	)
}

// Coherence scores how internally consistent the answer is.
func Coherence(opts ...Option) Metric {
	return newJudgeMetric(opts, NameCoherence,
		coherenceRequest, coherence.Parse,
		coherence.Score,
		func(o coherence.Output) string { return o.Reason },
	)
}

// Conciseness scores how succinct the answer is without losing meaning.
func Conciseness(opts ...Option) Metric {
	return newJudgeMetric(opts, NameConciseness,
		concisenessRequest, conciseness.Parse,
		conciseness.Score,
		func(o conciseness.Output) string { return o.Reason },
	)
}

// Completeness scores how completely the answer addresses the question.
func Completeness(opts ...Option) Metric {
	return newJudgeMetric(opts, NameCompleteness,
		completenessRequest, completeness.Parse,
		completeness.Score,
		func(o completeness.Output) string { return o.Reason },
	)
}

// InstructionAdherence scores how closely the answer followed
// Expected.Instructions. Requires Expected.Instructions.
func InstructionAdherence(opts ...Option) Metric {
	return newJudgeMetric(opts, NameInstructionAdherence,
		instructionAdherenceRequest, instructionadherence.Parse,
		instructionadherence.Score,
		func(o instructionadherence.Output) string { return o.Reason },
	)
}

// CitationAccuracy scores how accurate inline citations are against context.
func CitationAccuracy(opts ...Option) Metric {
	return newJudgeMetric(opts, NameCitationAccuracy,
		citationAccuracyRequest, citationaccuracy.Parse,
		func(o citationaccuracy.Output) float64 { return citationaccuracy.Score(o.Citations) },
		func(o citationaccuracy.Output) string { return o.Reason },
	)
}

// SummarizationQuality scores summary coverage, fidelity and conciseness as
// one score.
func SummarizationQuality(opts ...Option) Metric {
	return newJudgeMetric(opts, NameSummarizationQuality,
		summarizationQualityRequest, summarizationquality.Parse,
		func(o summarizationquality.Output) float64 {
			return summarizationquality.Score(o.CoverageScore, o.FidelityScore, o.ConcisenessScore)
		},
		func(o summarizationquality.Output) string { return o.Reason },
	)
}

// GEval scores along a free-form criteria string using the answer-relevancy
// schema. Criteria must be non-empty.
func GEval(criteria string, opts ...Option) Metric {
	criteria = strings.TrimSpace(criteria)
	if criteria == "" {
		return invalidMetric{name: NameGEval, err: optionErrorf("g-eval metric requires non-empty criteria")}
	}
	build := func(in EvalInput, name string) (JudgeRequest, error) {
		return JudgeRequest{
			Metric:       name,
			Question:     in.Input.Question,
			Answer:       in.Output.Answer,
			ContextDocs:  in.Input.Context,
			Instructions: prompt.GEvalInstructions(criteria),
			Schema:       prompt.AnswerRelevancySchema(),
		}, nil
	}
	return newJudgeMetric(opts, NameGEval, build,
		answerrelevancy.Parse,
		func(o answerrelevancy.Output) float64 { return o.Score },
		func(o answerrelevancy.Output) string { return o.Reason },
	)
}

// requestBuilder constructs a JudgeRequest from an EvalInput. It returns an
// error when the input lacks data required by the metric (e.g. Expected.Answer).
type requestBuilder func(in EvalInput, name string) (JudgeRequest, error)

func faithfulnessRequest(in EvalInput, name string) (JudgeRequest, error) {
	return JudgeRequest{
		Metric:       name,
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.FaithfulnessInstructions(),
		Schema:       prompt.FaithfulnessSchema(),
	}, nil
}

func answerRelevancyRequest(in EvalInput, name string) (JudgeRequest, error) {
	return JudgeRequest{
		Metric:       name,
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.AnswerRelevancyInstructions(),
		Schema:       prompt.AnswerRelevancySchema(),
	}, nil
}

func contextRelevancyRequest(in EvalInput, name string) (JudgeRequest, error) {
	return JudgeRequest{
		Metric:       name,
		Question:     in.Input.Question,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.ContextRelevancyInstructions(),
		Schema:       prompt.ContextRelevancySchema(),
	}, nil
}

func contextPrecisionRequest(in EvalInput, name string) (JudgeRequest, error) {
	return JudgeRequest{
		Metric:       name,
		Question:     in.Input.Question,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.ContextPrecisionInstructions(),
		Schema:       prompt.ContextPrecisionSchema(),
	}, nil
}

func contextRecallRequest(in EvalInput, name string) (JudgeRequest, error) {
	expected := strings.TrimSpace(in.Expected.Answer)
	if expected == "" {
		return JudgeRequest{}, requiresExpectedAnswerError(labelOf(name))
	}
	return JudgeRequest{
		Metric:         name,
		Question:       in.Input.Question,
		ExpectedAnswer: expected,
		ContextDocs:    in.Input.Context,
		Instructions:   prompt.ContextRecallInstructions(),
		Schema:         prompt.ContextRecallSchema(),
	}, nil
}

func answerCorrectnessRequest(in EvalInput, name string) (JudgeRequest, error) {
	expected := strings.TrimSpace(in.Expected.Answer)
	if expected == "" {
		return JudgeRequest{}, requiresExpectedAnswerError(labelOf(name))
	}
	return JudgeRequest{
		Metric:         name,
		Question:       in.Input.Question,
		Answer:         in.Output.Answer,
		ExpectedAnswer: expected,
		ContextDocs:    in.Input.Context,
		Instructions:   prompt.AnswerCorrectnessInstructions(),
		Schema:         prompt.AnswerCorrectnessSchema(),
	}, nil
}

func hallucinationRequest(in EvalInput, name string) (JudgeRequest, error) {
	return JudgeRequest{
		Metric:       name,
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.HallucinationInstructions(),
		Schema:       prompt.HallucinationSchema(),
	}, nil
}

func toxicityRequest(in EvalInput, name string) (JudgeRequest, error) {
	return JudgeRequest{
		Metric:       name,
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.ToxicityInstructions(),
		Schema:       prompt.ToxicitySchema(),
	}, nil
}

func biasRequest(in EvalInput, name string) (JudgeRequest, error) {
	return JudgeRequest{
		Metric:       name,
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.BiasInstructions(),
		Schema:       prompt.BiasSchema(),
	}, nil
}

func coherenceRequest(in EvalInput, name string) (JudgeRequest, error) {
	return JudgeRequest{
		Metric:       name,
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.CoherenceInstructions(),
		Schema:       prompt.CoherenceSchema(),
	}, nil
}

func concisenessRequest(in EvalInput, name string) (JudgeRequest, error) {
	return JudgeRequest{
		Metric:       name,
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.ConcisenessInstructions(),
		Schema:       prompt.ConcisenessSchema(),
	}, nil
}

func completenessRequest(in EvalInput, name string) (JudgeRequest, error) {
	return JudgeRequest{
		Metric:       name,
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.CompletenessInstructions(),
		Schema:       prompt.CompletenessSchema(),
	}, nil
}

func instructionAdherenceRequest(in EvalInput, name string) (JudgeRequest, error) {
	expected := strings.TrimSpace(in.Expected.Instructions)
	if expected == "" {
		return JudgeRequest{}, requiresExpectedInstructionsError(labelOf(name))
	}
	return JudgeRequest{
		Metric:               name,
		Question:             in.Input.Question,
		Answer:               in.Output.Answer,
		ExpectedInstructions: expected,
		ContextDocs:          in.Input.Context,
		Instructions:         prompt.InstructionAdherenceInstructions(),
		Schema:               prompt.InstructionAdherenceSchema(),
	}, nil
}

func citationAccuracyRequest(in EvalInput, name string) (JudgeRequest, error) {
	return JudgeRequest{
		Metric:       name,
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.CitationAccuracyInstructions(),
		Schema:       prompt.CitationAccuracySchema(),
	}, nil
}

func summarizationQualityRequest(in EvalInput, name string) (JudgeRequest, error) {
	return JudgeRequest{
		Metric:       name,
		Question:     in.Input.Question,
		Answer:       in.Output.Answer,
		ContextDocs:  in.Input.Context,
		Instructions: prompt.SummarizationQualityInstructions(),
		Schema:       prompt.SummarizationQualitySchema(),
	}, nil
}

// judgeMetric is the single generic implementation shared by every judge-based
// metric. The compiler specializes it per output type T at no runtime cost.
type judgeMetric[T any] struct {
	name     string
	cfg      config
	buildReq requestBuilder
	parse    func([]byte) (T, error)
	score    func(T) float64
	reason   func(T) string
}

func (m judgeMetric[T]) Name() string { return m.name }

func (m judgeMetric[T]) Evaluate(ctx context.Context, in EvalInput, j Judge) (Result, error) {
	label := labelOf(m.name)
	if j == nil {
		return Result{}, requiresJudgeError(label)
	}
	req, err := m.buildReq(in, m.name)
	if err != nil {
		return Result{}, err
	}
	resp, err := j.EvaluateJSON(ctx, req)
	if err != nil {
		return Result{}, judgeEvaluationError(err)
	}
	parsed, err := m.parse(resp.RawJSON)
	if err != nil {
		return Result{}, parseError(label, err)
	}
	details, err := json.Marshal(parsed)
	if err != nil {
		return Result{}, marshalError(label, err)
	}
	score := m.score(parsed)
	return Result{
		Name:         m.name,
		Score:        score,
		Pass:         score >= m.cfg.threshold,
		Reason:       m.reason(parsed),
		Details:      details,
		Provider:     resp.Provider,
		Model:        resp.Model,
		RequestID:    resp.RequestID,
		JudgeLatency: resp.Latency,
	}, nil
}

// newJudgeMetric applies options and returns the appropriate Metric: an
// invalidMetric on option error, or a fully wired judgeMetric[T] otherwise.
func newJudgeMetric[T any](
	opts []Option,
	name string,
	buildReq requestBuilder,
	parse func([]byte) (T, error),
	score func(T) float64,
	reason func(T) string,
) Metric {
	cfg, err := applyOptions(opts)
	if err != nil {
		return invalidMetric{name: name, err: err}
	}
	return judgeMetric[T]{
		name:     name,
		cfg:      cfg,
		buildReq: buildReq,
		parse:    parse,
		score:    score,
		reason:   reason,
	}
}
