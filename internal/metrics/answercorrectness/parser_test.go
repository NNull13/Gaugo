package answercorrectness

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"statements":[{"text":"x","correct":true,"reason":"supported"}],"score":0.8,"reason":"ok"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(out.Statements) != 1 || out.Statements[0].Text != "x" || !out.Statements[0].Correct {
		t.Fatalf("unexpected parsed statements: %+v", out.Statements)
	}
	if out.Score != 0.8 {
		t.Fatalf("score got=%f want=0.8", out.Score)
	}
	if out.Reason != "ok" {
		t.Fatalf("reason got=%q want=ok", out.Reason)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"statements":[],"score":0.7,"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected unknown-field error")
	}
	if _, err := Parse([]byte(`{"statements":[{"text":"x","correct":true,"reason":"ok","extra":1}],"score":0.7,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected nested unknown-field error")
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"score":0.7,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing statements error")
	}
	if _, err := Parse([]byte(`{"statements":[],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing score error")
	}
	if _, err := Parse([]byte(`{"statements":[],"score":0.7}`)); err == nil {
		t.Fatalf("expected missing reason error")
	}
	if _, err := Parse([]byte(`{"statements":[{"correct":true,"reason":"ok"}],"score":0.7,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing text error")
	}
	if _, err := Parse([]byte(`{"statements":[{"text":"x","reason":"ok"}],"score":0.7,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing correct error")
	}
	if _, err := Parse([]byte(`{"statements":[{"text":"x","correct":true}],"score":0.7,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing statement reason error")
	}
}

func TestParseOutOfRange(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"statements":[],"score":1.2,"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range error")
	}
	if _, err := Parse([]byte(`{"statements":[],"score":-0.1,"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range error")
	}
}

func TestScore(t *testing.T) {
	t.Parallel()

	if got := Score(Output{Score: 0.42}); got != 0.42 {
		t.Fatalf("score got=%f want=0.42", got)
	}
}
