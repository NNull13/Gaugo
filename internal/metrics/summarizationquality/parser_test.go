package summarizationquality

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"coverage_score":0.9,"fidelity_score":0.8,"conciseness_score":0.7,"reason":"ok"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if out.CoverageScore != 0.9 || out.FidelityScore != 0.8 || out.ConcisenessScore != 0.7 {
		t.Fatalf("unexpected parsed output: %+v", out)
	}
	if out.Reason != "ok" {
		t.Fatalf("reason got=%q want=ok", out.Reason)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"coverage_score":0.9,"fidelity_score":0.8,"conciseness_score":0.7,"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected unknown-field error")
	}
	if _, err := Parse([]byte(`{"coverage_score":0.9,"fidelity_score":0.8,"conciseness_score":0.7,"score":0.8,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected score to be rejected as an unknown field")
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"fidelity_score":0.8,"conciseness_score":0.7,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing coverage_score error")
	}
	if _, err := Parse([]byte(`{"coverage_score":0.9,"conciseness_score":0.7,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing fidelity_score error")
	}
	if _, err := Parse([]byte(`{"coverage_score":0.9,"fidelity_score":0.8,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing conciseness_score error")
	}
	if _, err := Parse([]byte(`{"coverage_score":0.9,"fidelity_score":0.8,"conciseness_score":0.7}`)); err == nil {
		t.Fatalf("expected missing reason error")
	}
}

func TestParseOutOfRange(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"coverage_score":1.2,"fidelity_score":0.8,"conciseness_score":0.7,"reason":"bad"}`)); err == nil {
		t.Fatalf("expected coverage_score out-of-range error")
	}
	if _, err := Parse([]byte(`{"coverage_score":0.9,"fidelity_score":-0.1,"conciseness_score":0.7,"reason":"bad"}`)); err == nil {
		t.Fatalf("expected fidelity_score out-of-range error")
	}
	if _, err := Parse([]byte(`{"coverage_score":0.9,"fidelity_score":0.8,"conciseness_score":1.1,"reason":"bad"}`)); err == nil {
		t.Fatalf("expected conciseness_score out-of-range error")
	}
}

func TestScore(t *testing.T) {
	t.Parallel()

	if got := Score(0.25, 0.5, 0.75); got != 0.5 {
		t.Fatalf("score got=%f want=0.5", got)
	}
}
