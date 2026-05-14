package request

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nnull13/gaugo"
)

func TestToEval(t *testing.T) {
	t.Parallel()

	in := gaugo.JudgeRequest{
		Metric:               "  Faithfulness ",
		Question:             "What is pricing?",
		Answer:               "Contact sales.",
		ExpectedAnswer:       "Enterprise customers contact sales.",
		ExpectedInstructions: "Answer tersely.",
		ContextDocs:          []gaugo.Document{{ID: "", Text: "enterprise via sales"}},
		Instructions:         "  Return JSON only  ",
		Schema:               json.RawMessage(`{"type":"object"}`),
	}
	got := ToEval(in)

	if got.Metric != "Faithfulness" {
		t.Fatalf("metric got=%q", got.Metric)
	}
	if got.Instructions != "Return JSON only" {
		t.Fatalf("instructions got=%q", got.Instructions)
	}
	if !strings.Contains(got.UserPrompt, "User Input:\nWhat is pricing?") {
		t.Fatalf("missing user input in user prompt: %q", got.UserPrompt)
	}
	if !strings.Contains(got.UserPrompt, "- [doc-1] enterprise via sales") {
		t.Fatalf("missing context in user prompt: %q", got.UserPrompt)
	}
	if !strings.Contains(got.UserPrompt, "Expected Answer:\nEnterprise customers contact sales.") {
		t.Fatalf("missing expected answer in user prompt: %q", got.UserPrompt)
	}
	if !strings.Contains(got.UserPrompt, "Expected Instructions:\nAnswer tersely.") {
		t.Fatalf("missing expected instructions in user prompt: %q", got.UserPrompt)
	}
}

func TestToRetryDefaults(t *testing.T) {
	t.Parallel()

	got := ToRetry(gaugo.RetryConfig{})
	if got.MaxAttempts != gaugo.DefaultRetryConfig().MaxAttempts {
		t.Fatalf("max attempts got=%d", got.MaxAttempts)
	}
}

func TestToRetryCustom(t *testing.T) {
	t.Parallel()

	got := ToRetry(gaugo.RetryConfig{
		MaxAttempts: 2,
		BaseDelay:   5 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	})
	if got.MaxAttempts != 2 || got.BaseDelay != 5*time.Millisecond || got.MaxDelay != 10*time.Millisecond {
		t.Fatalf("unexpected retry config: %+v", got)
	}
}
