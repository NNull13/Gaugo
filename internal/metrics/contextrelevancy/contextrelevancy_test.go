package contextrelevancy

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"documents":[{"id":"d1","score":0.8,"reason":"directly relevant"}],"reason":"ok"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(out.Documents) != 1 || out.Documents[0].ID != "d1" || out.Documents[0].Score != 0.8 {
		t.Fatalf("unexpected parsed documents: %+v", out.Documents)
	}
	if out.Reason != "ok" {
		t.Fatalf("reason got=%q want=ok", out.Reason)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"documents":[],"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected unknown-field error")
	}
	if _, err := Parse([]byte(`{"documents":[{"id":"d1","score":1,"reason":"ok","extra":1}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected nested unknown-field error")
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing documents error")
	}
	if _, err := Parse([]byte(`{"documents":[]}`)); err == nil {
		t.Fatalf("expected missing reason error")
	}
	if _, err := Parse([]byte(`{"documents":[{"score":1,"reason":"ok"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing id error")
	}
	if _, err := Parse([]byte(`{"documents":[{"id":"d1","reason":"ok"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing score error")
	}
	if _, err := Parse([]byte(`{"documents":[{"id":"d1","score":1}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing document reason error")
	}
}

func TestParseOutOfRange(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"documents":[{"id":"d1","score":1.2,"reason":"bad"}],"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range error")
	}
	if _, err := Parse([]byte(`{"documents":[{"id":"d1","score":-0.1,"reason":"bad"}],"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range error")
	}
}

func TestScore(t *testing.T) {
	t.Parallel()

	score := Score([]Document{
		{Score: 0.25},
		{Score: 1},
		{Score: 0.75},
	})
	if score != 2.0/3.0 {
		t.Fatalf("score got=%f", score)
	}
	if got := Score(nil); got != 0 {
		t.Fatalf("empty score got=%f want=0", got)
	}
}
