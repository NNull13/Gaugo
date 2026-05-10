package faithfulness

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"claims":[{"text":"x","supported":true}],"reason":"ok"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(out.Claims) != 1 || !out.Claims[0].Supported {
		t.Fatalf("unexpected parsed claims: %+v", out.Claims)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"claims":[],"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected error")
	}
}

func TestScore(t *testing.T) {
	t.Parallel()

	score := Score([]Claim{
		{Supported: true},
		{Supported: false},
		{Supported: true},
	})
	if score != 2.0/3.0 {
		t.Fatalf("score got=%f", score)
	}
	if got := Score(nil); got != 0 {
		t.Fatalf("empty score got=%f want=0", got)
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
	if _, err := Parse([]byte(`{"claims":[{"text":"x"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing supported error")
	}
}
