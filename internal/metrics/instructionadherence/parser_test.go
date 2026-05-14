package instructionadherence

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"instructions":[{"text":"use json","followed":true,"reason":"valid json"}],"reason":"all followed"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(out.Instructions) != 1 || out.Instructions[0].Text != "use json" || !out.Instructions[0].Followed || out.Instructions[0].Reason != "valid json" {
		t.Fatalf("unexpected parsed instructions: %+v", out.Instructions)
	}
	if out.Reason != "all followed" {
		t.Fatalf("reason got=%q want=all followed", out.Reason)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"instructions":[],"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected unknown-field error")
	}
	if _, err := Parse([]byte(`{"instructions":[],"score":1,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected score to be rejected as an unknown field")
	}
	if _, err := Parse([]byte(`{"instructions":[{"text":"x","followed":true,"reason":"ok","extra":1}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected nested unknown-field error")
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing instructions error")
	}
	if _, err := Parse([]byte(`{"instructions":[]}`)); err == nil {
		t.Fatalf("expected missing reason error")
	}
	if _, err := Parse([]byte(`{"instructions":[{"followed":true,"reason":"ok"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing instruction text error")
	}
	if _, err := Parse([]byte(`{"instructions":[{"text":"x","reason":"ok"}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing followed error")
	}
	if _, err := Parse([]byte(`{"instructions":[{"text":"x","followed":true}],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing instruction reason error")
	}
}

func TestScore(t *testing.T) {
	t.Parallel()

	score := Score(Output{Instructions: []Instruction{
		{Followed: true},
		{Followed: false},
		{Followed: true},
	}})
	if score != 2.0/3.0 {
		t.Fatalf("score got=%f", score)
	}
	if got := Score(Output{}); got != 0 {
		t.Fatalf("empty score got=%f want=0", got)
	}
}
