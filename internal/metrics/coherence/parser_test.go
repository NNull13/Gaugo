package coherence

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"score":0.8,"issues":[{"text":"jump","reason":"abrupt transition"}],"reason":"mostly coherent"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if out.Score != 0.8 {
		t.Fatalf("score got=%f want=0.8", out.Score)
	}
	if len(out.Issues) != 1 || out.Issues[0].Text != "jump" || out.Issues[0].Reason != "abrupt transition" {
		t.Fatalf("unexpected parsed issues: %+v", out.Issues)
	}
	if out.Reason != "mostly coherent" {
		t.Fatalf("reason got=%q want=mostly coherent", out.Reason)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"score":0.8,"issues":[],"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected unknown-field error")
	}
	if _, err := Parse([]byte(`{"score":0.8,"issues":[{"text":"x","reason":"ok","extra":1}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected nested unknown-field error")
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"issues":[],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing score error")
	}
	if _, err := Parse([]byte(`{"score":0.8,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing issues error")
	}
	if _, err := Parse([]byte(`{"score":0.8,"issues":[]}`)); err == nil {
		t.Fatalf("expected missing reason error")
	}
	if _, err := Parse([]byte(`{"score":0.8,"issues":[{"reason":"ok"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing issue text error")
	}
	if _, err := Parse([]byte(`{"score":0.8,"issues":[{"text":"x"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing issue reason error")
	}
}

func TestParseOutOfRange(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"score":1.2,"issues":[],"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range error")
	}
	if _, err := Parse([]byte(`{"score":-0.1,"issues":[],"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range error")
	}
}

func TestScore(t *testing.T) {
	t.Parallel()

	if got := Score(Output{Score: 0.75}); got != 0.75 {
		t.Fatalf("score got=%f want=0.75", got)
	}
}
