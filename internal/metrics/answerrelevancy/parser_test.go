package answerrelevancy

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	out, err := Parse([]byte(`{"score":0.8,"reason":"ok","issues":[]}`))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if out.Score != 0.8 {
		t.Fatalf("score got=%f want=0.8", out.Score)
	}
}

func TestParseOutOfRange(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"score":2,"reason":"bad"}`)); err == nil {
		t.Fatalf("expected out-of-range error")
	}
}

func TestParseUnknownField(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"score":0.7,"reason":"ok","extra":1}`)); err == nil {
		t.Fatalf("expected unknown-field error")
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]byte(`{"reason":"ok"}`)); err == nil {
		t.Fatalf("expected missing score error")
	}
	if _, err := Parse([]byte(`{"score":0}`)); err == nil {
		t.Fatalf("expected missing reason error")
	}
}
