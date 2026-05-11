package gaugo

import (
	"context"
	"testing"
)

func TestFaithfulnessMetricNoJudge(t *testing.T) {
	t.Parallel()

	_, err := Faithfulness().Evaluate(context.Background(), EvalInput{}, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestFaithfulnessMetricThreshold(t *testing.T) {
	t.Parallel()

	j := fakeJudge{
		eval: func(_ context.Context, _ JudgeRequest) (JudgeResponse, error) {
			return JudgeResponse{
				RawJSON: []byte(`{
					"claims":[
						{"text":"c1","supported":true},
						{"text":"c2","supported":false}
					],
					"reason":"partial"
				}`),
			}, nil
		},
	}
	res, err := Faithfulness(WithThreshold(0.4)).Evaluate(context.Background(), EvalInput{
		Input:  Input{Question: "Q"},
		Output: Output{Answer: "A"},
	}, j)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if !res.Pass {
		t.Fatalf("expected pass with threshold 0.4 and score %f", res.Score)
	}
}

func TestAnswerRelevancyMetricParseError(t *testing.T) {
	t.Parallel()

	j := fakeJudge{
		eval: func(_ context.Context, _ JudgeRequest) (JudgeResponse, error) {
			return JudgeResponse{RawJSON: []byte(`{"score":2,"reason":"bad"}`)}, nil
		},
	}
	_, err := AnswerRelevancy().Evaluate(context.Background(), EvalInput{
		Input:  Input{Question: "Q"},
		Output: Output{Answer: "A"},
	}, j)
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestContextRelevancyMetricScoresAverageAndIgnoresAnswer(t *testing.T) {
	t.Parallel()

	j := fakeJudge{
		eval: func(_ context.Context, req JudgeRequest) (JudgeResponse, error) {
			if req.Metric != "ContextRelevancy" {
				t.Fatalf("metric got=%q want=ContextRelevancy", req.Metric)
			}
			if req.Question != "Q" {
				t.Fatalf("question got=%q want=Q", req.Question)
			}
			if req.Answer != "" {
				t.Fatalf("context relevancy must not send answer to judge, got %q", req.Answer)
			}
			if len(req.ContextDocs) != 2 {
				t.Fatalf("context docs len=%d want=2", len(req.ContextDocs))
			}
			if len(req.Schema) == 0 {
				t.Fatalf("expected schema")
			}
			return JudgeResponse{
				RawJSON: []byte(`{
					"documents":[
						{"id":"d1","score":1,"reason":"direct"},
						{"id":"d2","score":0.2,"reason":"tangential"}
					],
					"reason":"mixed"
				}`),
			}, nil
		},
	}
	res, err := ContextRelevancy(WithThreshold(0.5)).Evaluate(context.Background(), EvalInput{
		Input: Input{
			Question: "Q",
			Context: []Document{
				{ID: "d1", Text: "direct"},
				{ID: "d2", Text: "tangential"},
			},
		},
		Output: Output{Answer: "A that must be ignored"},
	}, j)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if res.Name != "ContextRelevancy" {
		t.Fatalf("name got=%q want=ContextRelevancy", res.Name)
	}
	if res.Score != 0.6 {
		t.Fatalf("score got=%f want=0.6", res.Score)
	}
	if !res.Pass {
		t.Fatalf("expected pass with threshold 0.5 and score %f", res.Score)
	}
	if res.Reason != "mixed" {
		t.Fatalf("reason got=%q want=mixed", res.Reason)
	}
}

func TestContextRelevancyMetricEmptyDocumentsScoresZero(t *testing.T) {
	t.Parallel()

	j := fakeJudge{
		eval: func(_ context.Context, _ JudgeRequest) (JudgeResponse, error) {
			return JudgeResponse{RawJSON: []byte(`{"documents":[],"reason":"no context"}`)}, nil
		},
	}
	res, err := ContextRelevancy().Evaluate(context.Background(), EvalInput{
		Input: Input{Question: "Q"},
	}, j)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if res.Score != 0 {
		t.Fatalf("score got=%f want=0", res.Score)
	}
	if res.Pass {
		t.Fatalf("expected fail with default threshold and score 0")
	}
}

func TestContextRelevancyMetricParseError(t *testing.T) {
	t.Parallel()

	j := fakeJudge{
		eval: func(_ context.Context, _ JudgeRequest) (JudgeResponse, error) {
			return JudgeResponse{RawJSON: []byte(`{"documents":[{"id":"d1","score":2,"reason":"bad"}],"reason":"bad"}`)}, nil
		},
	}
	_, err := ContextRelevancy().Evaluate(context.Background(), EvalInput{
		Input: Input{Question: "Q"},
	}, j)
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestMetricInvalidThresholdFailsEvaluation(t *testing.T) {
	t.Parallel()

	_, err := AnswerRelevancy(WithThreshold(1.5)).Evaluate(context.Background(), EvalInput{}, fakeJudge{
		eval: func(context.Context, JudgeRequest) (JudgeResponse, error) {
			return JudgeResponse{RawJSON: []byte(`{"score":1,"reason":"ok"}`)}, nil
		},
	})
	if err == nil {
		t.Fatalf("expected invalid threshold error")
	}
}
