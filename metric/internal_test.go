package metric

import (
	"context"
	"encoding/json"
	"testing"
)

// fakeJudge is a stub Judge used by internal tests that need to exercise
// helper-backed code paths without spinning up a real LLM client.
type fakeJudge struct {
	eval func(ctx context.Context, req JudgeRequest) (JudgeResponse, error)
}

func (j fakeJudge) EvaluateJSON(ctx context.Context, req JudgeRequest) (JudgeResponse, error) {
	if j.eval == nil {
		return JudgeResponse{}, nil
	}
	return j.eval(ctx, req)
}

func TestMetricHelperEdgeCases(t *testing.T) {
	t.Parallel()

	if _, err := ContextPrecision().Evaluate(context.Background(), EvalInput{}, nil); err == nil {
		t.Fatalf("expected no-judge error from helper-backed metric")
	}
	if _, err := ContextPrecision().Evaluate(context.Background(), EvalInput{Input: Input{Question: "Q?"}}, fakeJudge{
		eval: func(context.Context, JudgeRequest) (JudgeResponse, error) {
			return JudgeResponse{RawJSON: []byte(`{"documents":[{"id":"d1","useful":true,"reason":"ok","extra":1}],"reason":"bad"}`)}, nil
		},
	}); err == nil {
		t.Fatalf("expected helper-backed parse error")
	}
	if _, err := deterministicResult("BadDetails", 0, 0.7, "bad", func() {}); err == nil {
		t.Fatalf("expected deterministic details marshal error")
	}
	if _, err := decodeJSONValue(`{} {}`, false); err == nil {
		t.Fatalf("expected trailing JSON value error")
	}
	if _, err := decodeJSONValue(`{} x`, false); err == nil {
		t.Fatalf("expected trailing JSON syntax error")
	}
	if _, err := normalizeJSONComparable(func() {}); err == nil {
		t.Fatalf("expected normalize JSON error")
	}
	if _, ok := lookupJSONPath(map[string]any{"items": []any{"x"}}, "items.nope"); ok {
		t.Fatalf("expected invalid array index lookup to fail")
	}
	if _, ok := lookupJSONPath(map[string]any{"items": []any{"x"}}, "items.9"); ok {
		t.Fatalf("expected out-of-range array lookup to fail")
	}
	if _, ok := lookupJSONPath("not-object", "x"); ok {
		t.Fatalf("expected scalar path lookup to fail")
	}
}

func TestSchemaComplianceTypeCoverage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		answer  string
		schema  string
		wantErr bool
	}{
		{name: "string", answer: `"Ada"`, schema: `{"type":"string"}`},
		{name: "boolean", answer: `true`, schema: `{"type":"boolean"}`},
		{name: "number", answer: `1.5`, schema: `{"type":"number"}`},
		{name: "number outside float64 range", answer: `1e400`, schema: `{"type":"number"}`},
		{name: "integer", answer: `2`, schema: `{"type":"integer"}`},
		{name: "large integer", answer: `9007199254740993`, schema: `{"type":"integer"}`},
		{name: "integer exponent", answer: `1.2e1`, schema: `{"type":"integer"}`},
		{name: "large integer exponent", answer: `1e400`, schema: `{"type":"integer"}`},
		{name: "integer mismatch", answer: `2.5`, schema: `{"type":"integer"}`, wantErr: true},
		{name: "large integer mismatch", answer: `9007199254740993.1`, schema: `{"type":"integer"}`, wantErr: true},
		{name: "negative exponent integer mismatch", answer: `1e-1`, schema: `{"type":"integer"}`, wantErr: true},
		{name: "null", answer: `null`, schema: `{"type":"null"}`},
		{name: "union", answer: `null`, schema: `{"type":["string","null"]}`},
		{name: "array items", answer: `["a","b"]`, schema: `{"type":"array","items":{"type":"string"}}`},
		{name: "array item mismatch", answer: `["a",1]`, schema: `{"type":"array","items":{"type":"string"}}`, wantErr: true},
		{name: "object additional property", answer: `{"name":"Ada","extra":true}`, schema: `{"type":"object","additionalProperties":false,"properties":{"name":{"type":"string"}}}`, wantErr: true},
		{name: "unknown type", answer: `"Ada"`, schema: `{"type":"mystery"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			value, err := decodeJSONValue(tt.answer, true)
			if err != nil {
				t.Fatalf("decodeJSONValue error: %v", err)
			}
			err = validateSchemaValue(value, json.RawMessage(tt.schema), "$")
			if tt.wantErr && err == nil {
				t.Fatalf("expected schema validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected schema validation error: %v", err)
			}
		})
	}
}

func TestSchemaTypeListValidation(t *testing.T) {
	t.Parallel()

	if got, err := schemaTypeList(nil); err != nil || got != nil {
		t.Fatalf("nil schema type got=%v err=%v", got, err)
	}
	if got, err := schemaTypeList("string"); err != nil || len(got) != 1 || got[0] != "string" {
		t.Fatalf("string schema type got=%v err=%v", got, err)
	}
	if got, err := schemaTypeList([]any{"string", "null"}); err != nil || len(got) != 2 {
		t.Fatalf("union schema type got=%v err=%v", got, err)
	}
	if _, err := schemaTypeList([]any{"string", 1}); err == nil {
		t.Fatalf("expected invalid union schema type error")
	}
	if _, err := schemaTypeList(1); err == nil {
		t.Fatalf("expected invalid schema type error")
	}
	if !matchesJSONType(float64(1.2), "number") {
		t.Fatalf("float64 should match number")
	}
	if !matchesJSONType(float64(2), "integer") {
		t.Fatalf("whole float64 should match integer")
	}
	if jsonTypeName(float64(1.2)) != "number" {
		t.Fatalf("float64 json type should be number")
	}
	if jsonTypeName(nil) != "null" {
		t.Fatalf("nil json type should be null")
	}
}

func TestLabelOfDerivesHumanReadable(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"Faithfulness":         "faithfulness",
		"AnswerRelevancy":      "answer relevancy",
		"ContextRelevancy":     "context relevancy",
		"ContextPrecision":     "context precision",
		"ContextRecall":        "context recall",
		"AnswerCorrectness":    "answer correctness",
		"Hallucination":        "hallucination",
		"Toxicity":             "toxicity",
		"Bias":                 "bias",
		"Coherence":            "coherence",
		"Conciseness":          "conciseness",
		"Completeness":         "completeness",
		"InstructionAdherence": "instruction adherence",
		"GEval":                "g eval",
		"CitationAccuracy":     "citation accuracy",
		"SummarizationQuality": "summarization quality",
		"ExpectedJSON":         "expected json",
		"AnswerSimilarity":     "answer similarity",
	}
	for name, want := range tests {
		if got := labelOf(name); got != want {
			t.Fatalf("labelOf(%q) got=%q want=%q", name, got, want)
		}
	}
}
