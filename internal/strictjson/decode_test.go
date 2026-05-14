package strictjson

import "testing"

func TestDecodeStrictOK(t *testing.T) {
	t.Parallel()

	var out struct {
		Name string `json:"name"`
	}
	if err := DecodeStrict([]byte(`{"name":"gaugo"}`), &out); err != nil {
		t.Fatalf("DecodeStrict error: %v", err)
	}
	if out.Name != "gaugo" {
		t.Fatalf("name got=%q want=%q", out.Name, "gaugo")
	}
}

func TestDecodeStrictUnknownField(t *testing.T) {
	t.Parallel()

	var out struct {
		Name string `json:"name"`
	}
	if err := DecodeStrict([]byte(`{"name":"gaugo","extra":1}`), &out); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDecodeStrictTrailingValue(t *testing.T) {
	t.Parallel()

	var out struct {
		Name string `json:"name"`
	}
	if err := DecodeStrict([]byte(`{"name":"gaugo"}{"x":1}`), &out); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDecodeStrictMalformed(t *testing.T) {
	t.Parallel()

	var out struct {
		Name string `json:"name"`
	}
	if err := DecodeStrict([]byte(`{"name":`), &out); err == nil {
		t.Fatalf("expected error")
	}
}
