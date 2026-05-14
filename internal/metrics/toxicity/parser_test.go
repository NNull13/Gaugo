package toxicity

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"toxic":true,"categories":[{"name":"insult","detected":true,"severity":0.7}],"score":0.7,"reason":"ok"}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if !out.Toxic || out.Score != 0.7 || len(out.Categories) != 1 {
		t.Fatalf("unexpected parsed output: %+v", out)
	}
	if out.Categories[0].Name != "insult" || !out.Categories[0].Detected || out.Categories[0].Severity != 0.7 {
		t.Fatalf("unexpected parsed categories: %+v", out.Categories)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"toxic":false,"categories":[],"score":0,"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected unknown-field error")
	}
	if _, err := Parse([]byte(`{"toxic":false,"categories":[{"name":"insult","detected":false,"severity":0,"extra":1}],"score":0,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected nested unknown-field error")
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"categories":[],"score":0,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing toxic error")
	}
	if _, err := Parse([]byte(`{"toxic":false,"score":0,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing categories error")
	}
	if _, err := Parse([]byte(`{"toxic":false,"categories":[],"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing score error")
	}
	if _, err := Parse([]byte(`{"toxic":false,"categories":[],"score":0}`)); err == nil {
		t.Fatalf("expected missing reason error")
	}
	if _, err := Parse([]byte(`{"toxic":false,"categories":[{"detected":false,"severity":0}],"score":0,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing name error")
	}
	if _, err := Parse([]byte(`{"toxic":false,"categories":[{"name":"insult","severity":0}],"score":0,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing detected error")
	}
	if _, err := Parse([]byte(`{"toxic":false,"categories":[{"name":"insult","detected":false}],"score":0,"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing severity error")
	}
}

func TestParseOutOfRange(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"toxic":true,"categories":[],"score":1.2,"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range score error")
	}
	if _, err := Parse([]byte(`{"toxic":true,"categories":[],"score":-0.1,"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range score error")
	}
	if _, err := Parse([]byte(`{"toxic":true,"categories":[{"name":"insult","detected":true,"severity":1.2}],"score":1,"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range severity error")
	}
	if _, err := Parse([]byte(`{"toxic":true,"categories":[{"name":"insult","detected":true,"severity":-0.1}],"score":1,"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range severity error")
	}
}

func TestScore(t *testing.T) {
	t.Parallel()

	out := Output{Score: 0.42}
	if got := Score(out); got != out.Score {
		t.Fatalf("score got=%f want=%f", got, out.Score)
	}
}
