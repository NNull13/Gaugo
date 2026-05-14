package citationaccuracy

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"citations":[{"text":"claim","doc_id":"doc-1","accurate":true,"reason":"supported"}],"reason":"ok"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(out.Citations) != 1 || out.Citations[0].Text != "claim" || out.Citations[0].DocID != "doc-1" || !out.Citations[0].Accurate {
		t.Fatalf("unexpected parsed citations: %+v", out.Citations)
	}
	if out.Reason != "ok" {
		t.Fatalf("reason got=%q want=ok", out.Reason)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"citations":[],"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected unknown-field error")
	}
	if _, err := Parse([]byte(`{"citations":[{"text":"claim","doc_id":"doc-1","accurate":true,"reason":"ok","extra":1}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected nested unknown-field error")
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing citations error")
	}
	if _, err := Parse([]byte(`{"citations":[]}`)); err == nil {
		t.Fatalf("expected missing reason error")
	}
	if _, err := Parse([]byte(`{"citations":[{"doc_id":"doc-1","accurate":true,"reason":"ok"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing text error")
	}
	if _, err := Parse([]byte(`{"citations":[{"text":"claim","accurate":true,"reason":"ok"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing doc_id error")
	}
	if _, err := Parse([]byte(`{"citations":[{"text":"claim","doc_id":"doc-1","reason":"ok"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing accurate error")
	}
	if _, err := Parse([]byte(`{"citations":[{"text":"claim","doc_id":"doc-1","accurate":true}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing citation reason error")
	}
}

func TestScore(t *testing.T) {
	t.Parallel()

	score := Score([]Citation{
		{Accurate: true},
		{Accurate: false},
		{Accurate: true},
	})
	if score != 2.0/3.0 {
		t.Fatalf("score got=%f", score)
	}
	if got := Score(nil); got != 0 {
		t.Fatalf("empty score got=%f want=0", got)
	}
}
