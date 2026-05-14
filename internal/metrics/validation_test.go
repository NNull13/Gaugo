package metrics

import (
	"strings"
	"testing"
)

func TestDecodeStrictWrapsDecodeErrors(t *testing.T) {
	t.Parallel()

	var out struct {
		Score float64 `json:"score"`
	}
	err := DecodeStrict([]byte(`{"score":1,"extra":true}`), &out, "parse metric")
	if err == nil {
		t.Fatalf("expected decode error")
	}
	if !strings.Contains(err.Error(), "parse metric:") {
		t.Fatalf("expected prefixed error, got %q", err)
	}
}

func TestMissingRequiredField(t *testing.T) {
	t.Parallel()

	err := MissingRequiredField("parse metric", "score")
	if got, want := err.Error(), "parse metric: missing required field score"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMissingRequiredNestedField(t *testing.T) {
	t.Parallel()

	err := MissingRequiredNestedField("parse metric", "claim", 2, "text")
	if got, want := err.Error(), "parse metric: claim 2 missing required field text"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRange01(t *testing.T) {
	t.Parallel()

	if err := Range01("parse metric", "score", 0.5); err != nil {
		t.Fatalf("expected score in range, got %v", err)
	}
	err := Range01("parse metric", "score", 1.2)
	if err == nil {
		t.Fatalf("expected range error")
	}
	if !strings.Contains(err.Error(), "score must be in [0,1]") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNestedRange01(t *testing.T) {
	t.Parallel()

	if err := NestedRange01("parse metric", "document", 0, "score", 0.5); err != nil {
		t.Fatalf("expected score in range, got %v", err)
	}
	err := NestedRange01("parse metric", "document", 0, "score", -0.1)
	if err == nil {
		t.Fatalf("expected range error")
	}
	if !strings.Contains(err.Error(), "document 0 score must be in [0,1]") {
		t.Fatalf("unexpected error: %v", err)
	}
}
