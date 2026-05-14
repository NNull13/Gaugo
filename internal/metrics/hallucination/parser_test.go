package hallucination

import (
	"math"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"claims":[{"text":"x","hallucinated":false,"reason":"supported"}],"reason":"ok"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(out.Claims) != 1 || out.Claims[0].Text != "x" || out.Claims[0].Hallucinated {
		t.Fatalf("unexpected parsed claims: %+v", out.Claims)
	}
	if out.Reason != "ok" {
		t.Fatalf("reason got=%q want=ok", out.Reason)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"claims":[],"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected unknown-field error")
	}
	if _, err := Parse([]byte(`{"claims":[{"text":"x","hallucinated":false,"reason":"ok","extra":1}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected nested unknown-field error")
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing claims error")
	}
	if _, err := Parse([]byte(`{"claims":[]}`)); err == nil {
		t.Fatalf("expected missing reason error")
	}
	if _, err := Parse([]byte(`{"claims":[{"hallucinated":false,"reason":"ok"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing text error")
	}
	if _, err := Parse([]byte(`{"claims":[{"text":"x","reason":"ok"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing hallucinated error")
	}
	if _, err := Parse([]byte(`{"claims":[{"text":"x","hallucinated":false}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing claim reason error")
	}
}

func TestScore(t *testing.T) {
	t.Parallel()

	score := Score([]Claim{
		{Hallucinated: false},
		{Hallucinated: true},
		{Hallucinated: false},
	})
	if math.Abs(score-2.0/3.0) > 1e-9 {
		t.Fatalf("score got=%f", score)
	}
	if got := Score(nil); got != 1 {
		t.Fatalf("empty score got=%f want=1", got)
	}
}
