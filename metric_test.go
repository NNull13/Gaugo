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
