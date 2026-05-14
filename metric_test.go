package gaugo

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
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

func TestLLMMetricConstructorsEvaluate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metric   Metric
		rawJSON  string
		input    EvalInput
		want     float64
		checkReq func(*testing.T, JudgeRequest)
	}{
		{
			name:   "ContextPrecision",
			metric: ContextPrecision(WithThreshold(0.5)),
			rawJSON: `{
				"documents":[
					{"id":"d1","useful":true,"reason":"needed"},
					{"id":"d2","useful":false,"reason":"noise"}
				],
				"reason":"mixed"
			}`,
			input: EvalInput{Input: Input{Question: "Q?", Context: []Document{Doc("d1", "needed"), Doc("d2", "noise")}}},
			want:  0.5,
			checkReq: func(t *testing.T, req JudgeRequest) {
				t.Helper()
				if req.Answer != "" || req.ExpectedAnswer != "" {
					t.Fatalf("context precision should ignore answers: %+v", req)
				}
			},
		},
		{
			name:   "ContextRecall",
			metric: ContextRecall(WithThreshold(0.5)),
			rawJSON: `{
				"claims":[
					{"text":"c1","attributed":true,"evidence":["d1"]},
					{"text":"c2","attributed":false}
				],
				"reason":"partial"
			}`,
			input: EvalInput{
				Input:    Input{Question: "Q?", Context: []Document{Doc("d1", "support")}},
				Expected: Expected{Answer: "ground truth"},
			},
			want: 0.5,
			checkReq: func(t *testing.T, req JudgeRequest) {
				t.Helper()
				if req.ExpectedAnswer != "ground truth" {
					t.Fatalf("ExpectedAnswer got=%q", req.ExpectedAnswer)
				}
			},
		},
		{
			name:   "AnswerCorrectness",
			metric: AnswerCorrectness(WithThreshold(0.5)),
			rawJSON: `{
				"statements":[{"text":"s1","correct":true,"reason":"matches"}],
				"score":0.8,
				"reason":"mostly correct"
			}`,
			input: EvalInput{
				Input:    Input{Question: "Q?"},
				Output:   Output{Answer: "actual"},
				Expected: Expected{Answer: "expected"},
			},
			want: 0.8,
			checkReq: func(t *testing.T, req JudgeRequest) {
				t.Helper()
				if req.Answer != "actual" || req.ExpectedAnswer != "expected" {
					t.Fatalf("answer correctness request mismatch: %+v", req)
				}
			},
		},
		{
			name:   "Hallucination",
			metric: Hallucination(WithThreshold(0.5)),
			rawJSON: `{
				"claims":[
					{"text":"c1","hallucinated":false,"reason":"supported"},
					{"text":"c2","hallucinated":true,"reason":"fabricated"}
				],
				"reason":"mixed"
			}`,
			input: EvalInput{Input: Input{Question: "Q?"}, Output: Output{Answer: "A"}},
			want:  0.5,
		},
		{
			name:   "Toxicity",
			metric: Toxicity(WithThreshold(0.5)),
			rawJSON: `{
				"toxic":false,
				"categories":[{"name":"hate_speech","detected":false,"severity":0}],
				"score":0.9,
				"reason":"safe"
			}`,
			input: EvalInput{Input: Input{Question: "Q?"}, Output: Output{Answer: "A"}},
			want:  0.9,
		},
		{
			name:    "Bias",
			metric:  Bias(WithThreshold(0.5)),
			rawJSON: `{"biased":false,"instances":[],"score":0.85,"reason":"fair"}`,
			input:   EvalInput{Input: Input{Question: "Q?"}, Output: Output{Answer: "A"}},
			want:    0.85,
		},
		{
			name:    "Coherence",
			metric:  Coherence(WithThreshold(0.5)),
			rawJSON: `{"score":0.75,"issues":[],"reason":"clear"}`,
			input:   EvalInput{Input: Input{Question: "Q?"}, Output: Output{Answer: "A"}},
			want:    0.75,
		},
		{
			name:    "Conciseness",
			metric:  Conciseness(WithThreshold(0.5)),
			rawJSON: `{"score":0.8,"issues":[],"reason":"brief"}`,
			input:   EvalInput{Input: Input{Question: "Q?"}, Output: Output{Answer: "A"}},
			want:    0.8,
		},
		{
			name:    "Completeness",
			metric:  Completeness(WithThreshold(0.5)),
			rawJSON: `{"score":0.6,"missing":[],"reason":"enough"}`,
			input:   EvalInput{Input: Input{Question: "Q?"}, Output: Output{Answer: "A"}},
			want:    0.6,
		},
		{
			name:   "InstructionAdherence",
			metric: InstructionAdherence(WithThreshold(0.5)),
			rawJSON: `{
				"instructions":[
					{"text":"json","followed":true,"reason":"ok"},
					{"text":"short","followed":false,"reason":"too long"}
				],
				"reason":"partial"
			}`,
			input: EvalInput{
				Input:    Input{Question: "Q?"},
				Output:   Output{Answer: "A"},
				Expected: Expected{Instructions: "Return JSON and be short."},
			},
			want: 0.5,
			checkReq: func(t *testing.T, req JudgeRequest) {
				t.Helper()
				if req.ExpectedInstructions != "Return JSON and be short." {
					t.Fatalf("ExpectedInstructions got=%q", req.ExpectedInstructions)
				}
			},
		},
		{
			name:    "GEval",
			metric:  GEval("prefer practical answers", WithThreshold(0.5)),
			rawJSON: `{"score":0.66,"reason":"ok","issues":[]}`,
			input:   EvalInput{Input: Input{Question: "Q?"}, Output: Output{Answer: "A"}},
			want:    0.66,
			checkReq: func(t *testing.T, req JudgeRequest) {
				t.Helper()
				if !strings.Contains(req.Instructions, "prefer practical answers") {
					t.Fatalf("criteria missing from instructions: %q", req.Instructions)
				}
			},
		},
		{
			name:   "CitationAccuracy",
			metric: CitationAccuracy(WithThreshold(0.5)),
			rawJSON: `{
				"citations":[
					{"text":"fact [d1]","doc_id":"d1","accurate":true,"reason":"supported"},
					{"text":"fact [d2]","doc_id":"d2","accurate":false,"reason":"wrong doc"}
				],
				"reason":"mixed"
			}`,
			input: EvalInput{Input: Input{Question: "Q?", Context: []Document{Doc("d1", "fact")}}, Output: Output{Answer: "A [d1]"}},
			want:  0.5,
		},
		{
			name:   "SummarizationQuality",
			metric: SummarizationQuality(WithThreshold(0.5)),
			rawJSON: `{
				"coverage_score":0.6,
				"fidelity_score":0.9,
				"conciseness_score":0.3,
				"reason":"mixed"
			}`,
			input: EvalInput{Input: Input{Question: "Summarize", Context: []Document{Doc("d1", "source")}}, Output: Output{Answer: "summary"}},
			want:  0.6,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			j := fakeJudge{
				eval: func(_ context.Context, req JudgeRequest) (JudgeResponse, error) {
					if req.Metric != tt.name {
						t.Fatalf("metric got=%q want=%q", req.Metric, tt.name)
					}
					if req.Question == "" {
						t.Fatalf("question should be forwarded")
					}
					if len(req.Schema) == 0 {
						t.Fatalf("schema should be set")
					}
					if strings.TrimSpace(req.Instructions) == "" {
						t.Fatalf("instructions should be set")
					}
					if tt.checkReq != nil {
						tt.checkReq(t, req)
					}
					return JudgeResponse{
						RawJSON:   []byte(tt.rawJSON),
						Provider:  "fake",
						Model:     "judge-v1",
						RequestID: "req-123",
						Latency:   10 * time.Millisecond,
					}, nil
				},
			}
			res, err := tt.metric.Evaluate(context.Background(), tt.input, j)
			if err != nil {
				t.Fatalf("Evaluate error: %v", err)
			}
			if res.Name != tt.name {
				t.Fatalf("result name got=%q want=%q", res.Name, tt.name)
			}
			if !floatNear(res.Score, tt.want) {
				t.Fatalf("score got=%f want=%f", res.Score, tt.want)
			}
			if !res.Pass {
				t.Fatalf("expected pass at threshold 0.5")
			}
			if len(res.Details) == 0 || !json.Valid(res.Details) {
				t.Fatalf("details should be valid JSON: %q", res.Details)
			}
			if res.Provider != "fake" || res.Model != "judge-v1" || res.RequestID != "req-123" || res.JudgeLatency != 10*time.Millisecond {
				t.Fatalf("judge metadata not propagated: %+v", res)
			}
		})
	}
}

func TestLLMMetricRequiresExpectedData(t *testing.T) {
	t.Parallel()

	for _, metric := range []Metric{ContextRecall(), AnswerCorrectness(), AnswerSimilarity()} {
		_, err := metric.Evaluate(context.Background(), EvalInput{}, fakeJudge{
			eval: func(context.Context, JudgeRequest) (JudgeResponse, error) {
				return JudgeResponse{RawJSON: []byte(`{}`)}, nil
			},
		})
		if err == nil || !strings.Contains(err.Error(), "Expected.Answer") {
			t.Fatalf("%s expected Expected.Answer error, got %v", metric.Name(), err)
		}
	}

	_, err := InstructionAdherence().Evaluate(context.Background(), EvalInput{}, fakeJudge{
		eval: func(context.Context, JudgeRequest) (JudgeResponse, error) {
			return JudgeResponse{RawJSON: []byte(`{}`)}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "Expected.Instructions") {
		t.Fatalf("expected Expected.Instructions error, got %v", err)
	}
}

func TestDeterministicJSONMetrics(t *testing.T) {
	t.Parallel()

	validJSON := EvalInput{Output: Output{Answer: `{"name":"Ada","meta":{"count":2},"items":[{"id":"x"}]}`}}

	res, err := JSONValidity().Evaluate(context.Background(), validJSON, nil)
	if err != nil {
		t.Fatalf("JSONValidity error: %v", err)
	}
	if !res.Pass || res.Score != 1 {
		t.Fatalf("JSONValidity pass result unexpected: %+v", res)
	}

	res, err = JSONValidity().Evaluate(context.Background(), EvalInput{Output: Output{Answer: `{bad`}}, nil)
	if err != nil {
		t.Fatalf("JSONValidity invalid error: %v", err)
	}
	if res.Pass || res.Score != 0 {
		t.Fatalf("JSONValidity invalid result unexpected: %+v", res)
	}

	schema := json.RawMessage(`{
		"type":"object",
		"additionalProperties":false,
		"required":["name","meta","items"],
		"properties":{
			"name":{"type":"string"},
			"meta":{
				"type":"object",
				"required":["count"],
				"properties":{"count":{"type":"integer"}}
			},
			"items":{"type":"array","items":{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}}}
		}
	}`)
	res, err = SchemaCompliance(WithSchema(schema)).Evaluate(context.Background(), validJSON, nil)
	if err != nil {
		t.Fatalf("SchemaCompliance error: %v", err)
	}
	if !res.Pass || res.Score != 1 {
		t.Fatalf("SchemaCompliance pass result unexpected: %+v", res)
	}

	res, err = SchemaCompliance(WithSchema(schema)).Evaluate(context.Background(), EvalInput{Output: Output{Answer: `{"name":"Ada","meta":{},"items":[]}`}}, nil)
	if err != nil {
		t.Fatalf("SchemaCompliance mismatch error: %v", err)
	}
	if res.Pass || !strings.Contains(res.Reason, "missing required field") {
		t.Fatalf("SchemaCompliance mismatch unexpected: %+v", res)
	}

	res, err = ExpectedJSON(WithExpectedFields(map[string]any{
		"name":       "Ada",
		"meta.count": 2,
		"items.0.id": "x",
	})).Evaluate(context.Background(), validJSON, nil)
	if err != nil {
		t.Fatalf("ExpectedJSON error: %v", err)
	}
	if !res.Pass || res.Score != 1 {
		t.Fatalf("ExpectedJSON pass result unexpected: %+v", res)
	}

	res, err = ExpectedJSON(WithExpectedFields(map[string]any{
		"name":       "Ada",
		"meta.count": 3,
	})).Evaluate(context.Background(), validJSON, nil)
	if err != nil {
		t.Fatalf("ExpectedJSON mismatch error: %v", err)
	}
	if res.Pass || res.Score != 0.5 {
		t.Fatalf("ExpectedJSON mismatch result unexpected: %+v", res)
	}
}

func TestExpectedJSONExactNumericComparison(t *testing.T) {
	t.Parallel()

	metric := ExpectedJSON(WithExpectedFields(map[string]any{
		"big":     json.Number("9007199254740993"),
		"decimal": json.Number("1.2300"),
		"exp":     json.Number("1e3"),
	}))

	res, err := metric.Evaluate(context.Background(), EvalInput{
		Output: Output{Answer: `{"big":9007199254740993,"decimal":1.23,"exp":1000}`},
	}, nil)
	if err != nil {
		t.Fatalf("ExpectedJSON error: %v", err)
	}
	if !res.Pass || res.Score != 1 {
		t.Fatalf("ExpectedJSON exact numeric pass result unexpected: %+v", res)
	}

	res, err = ExpectedJSON(WithExpectedFields(map[string]any{
		"big":     json.Number("9007199254740992"),
		"decimal": json.Number("1.2301"),
	})).Evaluate(context.Background(), EvalInput{
		Output: Output{Answer: `{"big":9007199254740993,"decimal":1.23}`},
	}, nil)
	if err != nil {
		t.Fatalf("ExpectedJSON mismatch error: %v", err)
	}
	if res.Pass || res.Score != 0 {
		t.Fatalf("ExpectedJSON exact numeric mismatch result unexpected: %+v", res)
	}
}

func TestDeterministicAnswerMetrics(t *testing.T) {
	t.Parallel()

	res, err := AnswerSimilarity(WithThreshold(0.6)).Evaluate(context.Background(), EvalInput{
		Output:   Output{Answer: "hello world"},
		Expected: Expected{Answer: "hello brave world"},
	}, nil)
	if err != nil {
		t.Fatalf("AnswerSimilarity error: %v", err)
	}
	if !res.Pass || !floatNear(res.Score, 2.0/3.0) {
		t.Fatalf("AnswerSimilarity result unexpected: %+v", res)
	}

	res, err = Latency(WithMaxLatency(100*time.Millisecond)).Evaluate(context.Background(), EvalInput{
		Elapsed: 50 * time.Millisecond,
	}, nil)
	if err != nil {
		t.Fatalf("Latency error: %v", err)
	}
	if !res.Pass || res.Score != 1 {
		t.Fatalf("Latency pass result unexpected: %+v", res)
	}

	res, err = Latency(WithMaxLatency(100*time.Millisecond)).Evaluate(context.Background(), EvalInput{
		Elapsed: 150 * time.Millisecond,
	}, nil)
	if err != nil {
		t.Fatalf("Latency failure error: %v", err)
	}
	if res.Pass || res.Score != 0 {
		t.Fatalf("Latency failure result unexpected: %+v", res)
	}

	res, err = AnswerLength(WithMinLength(2), WithMaxLength(4)).Evaluate(context.Background(), EvalInput{
		Output: Output{Answer: "abcd"},
	}, nil)
	if err != nil {
		t.Fatalf("AnswerLength error: %v", err)
	}
	if !res.Pass || res.Score != 1 {
		t.Fatalf("AnswerLength pass result unexpected: %+v", res)
	}

	res, err = ExpectedRegex(`(?i)^hello`).Evaluate(context.Background(), EvalInput{
		Output: Output{Answer: "Hello world"},
	}, nil)
	if err != nil {
		t.Fatalf("ExpectedRegex error: %v", err)
	}
	if !res.Pass || res.Score != 1 {
		t.Fatalf("ExpectedRegex pass result unexpected: %+v", res)
	}

	res, err = ExpectedRegex(`^bye`).Evaluate(context.Background(), EvalInput{
		Output: Output{Answer: "Hello world"},
	}, nil)
	if err != nil {
		t.Fatalf("ExpectedRegex failure error: %v", err)
	}
	if res.Pass || res.Score != 0 {
		t.Fatalf("ExpectedRegex failure result unexpected: %+v", res)
	}
}

func TestDeterministicMetricInvalidOptions(t *testing.T) {
	t.Parallel()

	cases := []Metric{
		SchemaCompliance(),
		SchemaCompliance(WithSchema(json.RawMessage(`{bad`))),
		ExpectedJSON(),
		ExpectedJSON(WithExpectedFields(map[string]any{})),
		JSONValidity(WithThreshold(math.NaN())),
		JSONValidity(WithThreshold(math.Inf(1))),
		JSONValidity(WithThreshold(math.Inf(-1))),
		Latency(),
		Latency(WithMaxLatency(-time.Millisecond)),
		AnswerLength(),
		AnswerLength(WithMinLength(5), WithMaxLength(2)),
		ExpectedRegex(""),
		ExpectedRegex(" \t\n "),
		ExpectedRegex(`[`),
		GEval(""),
	}
	for _, metric := range cases {
		_, err := metric.Evaluate(context.Background(), EvalInput{}, nil)
		if err == nil {
			t.Fatalf("%s expected invalid metric error", metric.Name())
		}
	}
}

func floatNear(got, want float64) bool {
	const epsilon = 0.0000001
	if got < want-epsilon || got > want+epsilon {
		return false
	}
	return true
}
