package completeness

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"score":0.8,"missing":[{"aspect":"edge cases","reason":"not discussed"}],"reason":"mostly complete"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if out.Score != 0.8 {
		t.Fatalf("score got=%f want=0.8", out.Score)
	}
	if len(out.Missing) != 1 || out.Missing[0].Aspect != "edge cases" || out.Missing[0].Reason != "not discussed" {
		t.Fatalf("unexpected parsed missing items: %+v", out.Missing)
	}
	if out.Reason != "mostly complete" {
		t.Fatalf("reason got=%q want=mostly complete", out.Reason)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"score":0.8,"missing":[],"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected unknown-field error")
	}
	if _, err := Parse([]byte(`{"score":0.8,"missing":[{"aspect":"x","reason":"ok","extra":1}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected nested unknown-field error")
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"missing":[],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing score error")
	}
	if _, err := Parse([]byte(`{"score":0.8,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing missing error")
	}
	if _, err := Parse([]byte(`{"score":0.8,"missing":[]}`)); err == nil {
		t.Fatalf("expected missing reason error")
	}
	if _, err := Parse([]byte(`{"score":0.8,"missing":[{"reason":"ok"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing aspect error")
	}
	if _, err := Parse([]byte(`{"score":0.8,"missing":[{"aspect":"x"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing item reason error")
	}
}

func TestParseOutOfRange(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"score":1.2,"missing":[],"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range error")
	}
	if _, err := Parse([]byte(`{"score":-0.1,"missing":[],"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range error")
	}
}

func TestScore(t *testing.T) {
	t.Parallel()

	if got := Score(Output{Score: 0.75}); got != 0.75 {
		t.Fatalf("score got=%f want=0.75", got)
	}
}
