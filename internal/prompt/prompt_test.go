package prompt

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildUserPromptNoDocs(t *testing.T) {
	t.Parallel()

	got := BuildUserPrompt("  Q?  ", "  A.  ", nil)
	if !strings.Contains(got, "DATA-BEGIN USER_INPUT\n") {
		t.Fatalf("missing user input data frame: %q", got)
	}
	if !strings.Contains(got, "User Input:\nQ?") {
		t.Fatalf("missing trimmed user input in prompt: %q", got)
	}
	if !strings.Contains(got, "Actual Output:\nA.") {
		t.Fatalf("missing trimmed actual output in prompt: %q", got)
	}
	if !strings.Contains(got, "Reference Context:\n(none)") {
		t.Fatalf("missing no-context marker: %q", got)
	}
}

func TestBuildUserPromptWithDocs(t *testing.T) {
	t.Parallel()

	got := BuildUserPrompt("Q", "A", []Document{
		{ID: "", Text: "  first  "},
		{ID: "doc-x", Text: "second"},
		{ID: "doc-empty", Text: "   "},
	})
	if !strings.Contains(got, "- [doc-1] first") {
		t.Fatalf("missing auto-generated id doc line: %q", got)
	}
	if !strings.Contains(got, "- [doc-x] second") {
		t.Fatalf("missing explicit id doc line: %q", got)
	}
	if strings.Contains(got, "doc-empty") {
		t.Fatalf("empty context document should be omitted: %q", got)
	}
}

func TestBuildUserPromptAllEmptyDocs(t *testing.T) {
	t.Parallel()

	got := BuildUserPrompt("Q", "A", []Document{
		{ID: "doc-a", Text: "   "},
		{ID: "", Text: ""},
	})
	if !strings.Contains(got, "Reference Context:\n(none)") {
		t.Fatalf("expected none marker when docs are empty: %q", got)
	}
}

func TestBuildUserPromptDocCapAndTruncationMarker(t *testing.T) {
	t.Parallel()

	text := strings.Repeat("x", maxDocBytes+128)
	got := BuildUserPrompt("Q", "A", []Document{{ID: "doc-x", Text: text}})
	wantDoc := truncateText(text, maxDocBytes)
	if !strings.Contains(got, "- [doc-x] "+wantDoc) {
		t.Fatalf("doc text was not capped as expected")
	}
	if !strings.Contains(wantDoc, truncationMarker) {
		t.Fatalf("capped doc text should contain truncation marker")
	}
}

func TestBuildUserPromptPromptCapAndTruncationMarker(t *testing.T) {
	t.Parallel()

	large := strings.Repeat("q", maxPromptBytes)
	got := BuildUserPrompt(large, large, []Document{{ID: "doc-1", Text: large}})

	if len(got) > maxPromptBytes {
		t.Fatalf("prompt length exceeded max bytes: got=%d limit=%d", len(got), maxPromptBytes)
	}
	if !strings.Contains(got, truncationMarker) {
		t.Fatalf("expected prompt truncation marker")
	}
}

func TestMetricInstructionsContainStrictOutputRules(t *testing.T) {
	t.Parallel()

	faithfulness := FaithfulnessInstructions()
	if !strings.Contains(strings.ToLower(faithfulness), "return valid json only") {
		t.Fatalf("faithfulness instructions must enforce raw json output: %q", faithfulness)
	}
	if strings.Contains(strings.ToLower(faithfulness), "rag") {
		t.Fatalf("faithfulness instructions must stay generic: %q", faithfulness)
	}
	if !strings.Contains(faithfulness, "supported=false") {
		t.Fatalf("faithfulness instructions must define unsupported behavior: %q", faithfulness)
	}
	if !strings.Contains(faithfulness, "self-contained atomic factual claims") {
		t.Fatalf("faithfulness instructions must require atomic claim extraction: %q", faithfulness)
	}

	relevancy := AnswerRelevancyInstructions()
	if !strings.Contains(relevancy, "any decimal in [0,1]") {
		t.Fatalf("answer relevancy instructions must define score range: %q", relevancy)
	}
	if !strings.Contains(relevancy, "full continuous range") {
		t.Fatalf("answer relevancy instructions must avoid discrete-only scoring: %q", relevancy)
	}
	if !strings.Contains(relevancy, "Mentally break the output into statements") {
		t.Fatalf("answer relevancy instructions must evaluate statement-level relevance: %q", relevancy)
	}
	if !strings.Contains(strings.ToLower(relevancy), "return valid json only") {
		t.Fatalf("answer relevancy instructions must enforce raw json output: %q", relevancy)
	}
}

func TestSchemasAreValidJSON(t *testing.T) {
	t.Parallel()

	for _, raw := range []json.RawMessage{FaithfulnessSchema(), AnswerRelevancySchema()} {
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("schema is not valid json: %v", err)
		}
	}
}
