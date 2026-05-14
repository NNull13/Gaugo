package contextrecall

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"claims":[{"text":"x","attributed":true,"evidence":["d1"]}],"reason":"ok"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(out.Claims) != 1 || out.Claims[0].Text != "x" || !out.Claims[0].Attributed {
		t.Fatalf("unexpected parsed claims: %+v", out.Claims)
	}
	if len(out.Claims[0].Evidence) != 1 || out.Claims[0].Evidence[0] != "d1" {
		t.Fatalf("unexpected parsed evidence: %+v", out.Claims[0].Evidence)
	}
	if out.Reason != "ok" {
		t.Fatalf("reason got=%q want=ok", out.Reason)
	}
}

func TestParseAllowsMissingEvidence(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"claims":[{"text":"x","attributed":false}],"reason":"ok"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(out.Claims) != 1 || out.Claims[0].Evidence != nil {
		t.Fatalf("unexpected parsed claims: %+v", out.Claims)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"claims":[],"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected unknown-field error")
	}
	if _, err := Parse([]byte(`{"claims":[{"text":"x","attributed":true,"extra":1}],"reason":"ok"}`)); err == nil {
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
	if _, err := Parse([]byte(`{"claims":[{"attributed":true}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing text error")
	}
	if _, err := Parse([]byte(`{"claims":[{"text":"x"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing attributed error")
	}
}

func TestScore(t *testing.T) {
	t.Parallel()

	score := Score([]Claim{
		{Attributed: true},
		{Attributed: false},
		{Attributed: true},
	})
	if score != 2.0/3.0 {
		t.Fatalf("score got=%f", score)
	}
	if got := Score(nil); got != 0 {
		t.Fatalf("empty score got=%f want=0", got)
	}
}
