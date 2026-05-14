package bias

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"biased":true,"instances":[{"text":"x","bias_type":"stereotype","reason":"ok"}],"score":0.6,"reason":"ok"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if !out.Biased || out.Score != 0.6 || len(out.Instances) != 1 {
		t.Fatalf("unexpected parsed output: %+v", out)
	}
	if out.Instances[0].Text != "x" || out.Instances[0].BiasType != "stereotype" {
		t.Fatalf("unexpected parsed instances: %+v", out.Instances)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"biased":false,"instances":[],"score":0,"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected unknown-field error")
	}
	if _, err := Parse([]byte(`{"biased":false,"instances":[{"text":"x","bias_type":"none","reason":"ok","extra":1}],"score":0,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected nested unknown-field error")
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"instances":[],"score":0,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing biased error")
	}
	if _, err := Parse([]byte(`{"biased":false,"score":0,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing instances error")
	}
	if _, err := Parse([]byte(`{"biased":false,"instances":[],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing score error")
	}
	if _, err := Parse([]byte(`{"biased":false,"instances":[],"score":0}`)); err == nil {
		t.Fatalf("expected missing reason error")
	}
	if _, err := Parse([]byte(`{"biased":false,"instances":[{"bias_type":"none","reason":"ok"}],"score":0,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing text error")
	}
	if _, err := Parse([]byte(`{"biased":false,"instances":[{"text":"x","reason":"ok"}],"score":0,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing bias_type error")
	}
	if _, err := Parse([]byte(`{"biased":false,"instances":[{"text":"x","bias_type":"none"}],"score":0,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing instance reason error")
	}
}

func TestParseOutOfRange(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"biased":true,"instances":[],"score":1.2,"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range score error")
	}
	if _, err := Parse([]byte(`{"biased":true,"instances":[],"score":-0.1,"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range score error")
	}
}

func TestScore(t *testing.T) {
	t.Parallel()

	out := Output{Score: 0.35}
	if got := Score(out); got != out.Score {
		t.Fatalf("score got=%f want=%f", got, out.Score)
	}
}
