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

	contextRelevancy := ContextRelevancyInstructions()
	if !strings.Contains(contextRelevancy, "Ignore the actual output entirely") {
		t.Fatalf("context relevancy instructions must ignore actual output: %q", contextRelevancy)
	}
	if !strings.Contains(contextRelevancy, "each reference context document") {
		t.Fatalf("context relevancy instructions must evaluate documents individually: %q", contextRelevancy)
	}
	if !strings.Contains(contextRelevancy, "any decimal in [0,1]") {
		t.Fatalf("context relevancy instructions must define score range: %q", contextRelevancy)
	}
	if !strings.Contains(strings.ToLower(contextRelevancy), "return valid json only") {
		t.Fatalf("context relevancy instructions must enforce raw json output: %q", contextRelevancy)
	}
}

func TestSchemasAreValidJSON(t *testing.T) {
	t.Parallel()

	for _, raw := range []json.RawMessage{FaithfulnessSchema(), AnswerRelevancySchema(), ContextRelevancySchema()} {
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("schema is not valid json: %v", err)
		}
	}
}

// --- cappedWriter tests ---

func TestCappedWriterEmptyString(t *testing.T) {
	t.Parallel()

	var w cappedWriter
	w.limit = 100
	w.writeString("")
	if w.String() != "" {
		t.Fatalf("writing empty string should produce empty output")
	}
	if w.isTruncated() {
		t.Fatalf("should not be truncated after empty write")
	}
}

func TestCappedWriterZeroLimit(t *testing.T) {
	t.Parallel()

	var w cappedWriter
	w.limit = 0
	w.writeString("hello")
	if w.String() != "" {
		t.Fatalf("zero limit should produce empty output, got %q", w.String())
	}
	if w.isTruncated() {
		t.Fatalf("zero limit should not set truncated (early return)")
	}
}

func TestCappedWriterAlreadyTruncated(t *testing.T) {
	t.Parallel()

	var w cappedWriter
	w.limit = 10
	w.truncated = true
	w.writeString("hello")
	if w.String() != "" {
		t.Fatalf("already truncated writer should not accept more data")
	}
}

func TestCappedWriterRemainingZero(t *testing.T) {
	t.Parallel()

	// Fill buffer exactly to limit, then write more to trigger remaining <= 0.
	var w cappedWriter
	w.limit = 5
	w.buf.WriteString("12345") // fill exactly
	w.writeString("more")
	if !w.isTruncated() {
		t.Fatalf("should be truncated when remaining is zero")
	}
	// The output should have the truncation marker appended via markTruncated.
	// Since buf was already at limit, markTruncated must reset and rewrite.
	got := w.String()
	if len(got) > 5 {
		t.Fatalf("output should not exceed limit, got len=%d", len(got))
	}
}

func TestCappedWriterKeepNegative(t *testing.T) {
	t.Parallel()

	// Set limit small enough that remaining < len(truncationMarker).
	// This triggers keep < 0 => keep = 0 in writeString.
	markerLen := len(truncationMarker)
	var w cappedWriter
	w.limit = markerLen + 2 // small limit
	// Write enough to leave remaining < markerLen
	w.buf.WriteString(strings.Repeat("a", 3))
	// Now remaining = markerLen + 2 - 3 = markerLen - 1, which is < markerLen
	// So keep = remaining - markerLen < 0
	w.writeString(strings.Repeat("b", 100))
	if !w.isTruncated() {
		t.Fatalf("should be truncated")
	}
	if len(w.String()) > w.limit {
		t.Fatalf("output exceeded limit: got=%d limit=%d", len(w.String()), w.limit)
	}
}

func TestCappedWriterMarkTruncatedResetAndRewrite(t *testing.T) {
	t.Parallel()

	// Fill buffer close to limit, then trigger truncation that requires
	// markTruncated to reset the buffer and rewrite with prefix + marker.
	markerLen := len(truncationMarker)
	limit := markerLen + 10
	var w cappedWriter
	w.limit = limit
	// Fill buffer with content that plus marker exceeds limit.
	w.buf.WriteString(strings.Repeat("x", limit-2))
	// Now buf.Len() = limit-2, adding marker would exceed limit.
	w.markTruncated()
	if !w.isTruncated() {
		t.Fatalf("should be truncated")
	}
	got := w.String()
	if len(got) > limit {
		t.Fatalf("markTruncated output exceeded limit: got=%d limit=%d", len(got), limit)
	}
	if !strings.HasSuffix(got, truncationMarker) {
		t.Fatalf("output should end with truncation marker, got %q", got)
	}
}

func TestCappedWriterMarkTruncatedMarkerExceedsLimit(t *testing.T) {
	t.Parallel()

	// When the truncation marker itself is larger than the limit,
	// markTruncated should reset and write a prefix of the marker.
	var w cappedWriter
	w.limit = 3 // much smaller than len(truncationMarker)
	w.buf.WriteString("ab")
	w.markTruncated()
	if !w.isTruncated() {
		t.Fatalf("should be truncated")
	}
	got := w.String()
	if len(got) > 3 {
		t.Fatalf("output exceeded limit: got=%d", len(got))
	}
	want := truncationMarker[:3]
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCappedWriterMarkTruncatedMarkerFits(t *testing.T) {
	t.Parallel()

	// When there's enough room to just append the marker.
	markerLen := len(truncationMarker)
	var w cappedWriter
	w.limit = markerLen + 50
	w.buf.WriteString("hello")
	w.markTruncated()
	if !w.isTruncated() {
		t.Fatalf("should be truncated")
	}
	got := w.String()
	if got != "hello"+truncationMarker {
		t.Fatalf("expected marker appended, got %q", got)
	}
}

func TestMarkTruncatedIdempotent(t *testing.T) {
	t.Parallel()

	var w cappedWriter
	w.limit = 100
	w.writeString("hello")
	w.markTruncated()
	first := w.String()
	w.markTruncated() // second call should be no-op
	second := w.String()
	if first != second {
		t.Fatalf("markTruncated should be idempotent: %q != %q", first, second)
	}
}

func TestMarkTruncatedZeroLimit(t *testing.T) {
	t.Parallel()

	var w cappedWriter
	w.limit = 0
	w.markTruncated()
	if w.isTruncated() {
		t.Fatalf("markTruncated with zero limit should be no-op")
	}
}

// --- prefixByBytes tests ---

func TestPrefixByBytesEmpty(t *testing.T) {
	t.Parallel()

	if got := prefixByBytes("", 10); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestPrefixByBytesZeroLimit(t *testing.T) {
	t.Parallel()

	if got := prefixByBytes("hello", 0); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestPrefixByBytesNegativeLimit(t *testing.T) {
	t.Parallel()

	if got := prefixByBytes("hello", -5); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestPrefixByBytesNoTruncation(t *testing.T) {
	t.Parallel()

	if got := prefixByBytes("abc", 10); got != "abc" {
		t.Fatalf("expected full string, got %q", got)
	}
}

func TestPrefixByBytesExactFit(t *testing.T) {
	t.Parallel()

	if got := prefixByBytes("abc", 3); got != "abc" {
		t.Fatalf("expected full string, got %q", got)
	}
}

func TestPrefixByBytesASCIICut(t *testing.T) {
	t.Parallel()

	if got := prefixByBytes("abcdef", 3); got != "abc" {
		t.Fatalf("expected 'abc', got %q", got)
	}
}

func TestPrefixByBytesMultibyteUTF8(t *testing.T) {
	t.Parallel()

	// U+1F600 (grinning face) is 4 bytes in UTF-8.
	emoji := "\U0001F600"
	// Cutting at 2 bytes should skip back past continuation bytes, yielding empty.
	got := prefixByBytes(emoji+"abc", 2)
	if strings.Contains(got, "\xF0") {
		t.Fatalf("should not contain partial rune bytes, got %q", got)
	}
	// With limit=1, can't fit even the first byte of the emoji as a valid rune.
	got2 := prefixByBytes(emoji, 1)
	if got2 != "" {
		t.Fatalf("partial emoji should yield empty, got %q", got2)
	}
}

func TestPrefixByBytesCJK(t *testing.T) {
	t.Parallel()

	// CJK character U+4E16 ("world") is 3 bytes in UTF-8.
	cjk := "\u4E16\u754C" // two 3-byte chars = 6 bytes
	// Cut at 4 bytes: should keep only the first char (3 bytes).
	got := prefixByBytes(cjk, 4)
	if got != "\u4E16" {
		t.Fatalf("expected single CJK char, got %q (len=%d)", got, len(got))
	}
}

func TestPrefixByBytesTwoBytRune(t *testing.T) {
	t.Parallel()

	// U+00E9 (e with accent) is 2 bytes.
	s := "\u00E9\u00E9\u00E9" // 6 bytes
	got := prefixByBytes(s, 3)
	if got != "\u00E9" {
		t.Fatalf("expected one 2-byte rune, got %q (len=%d)", got, len(got))
	}
}

// --- copyPrefixByBytes tests ---

func TestCopyPrefixByBytesEmpty(t *testing.T) {
	t.Parallel()

	got := copyPrefixByBytes(nil, 10)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
	got = copyPrefixByBytes([]byte{}, 10)
	if got != nil {
		t.Fatalf("expected nil for empty slice, got %v", got)
	}
}

func TestCopyPrefixByBytesZeroLimit(t *testing.T) {
	t.Parallel()

	got := copyPrefixByBytes([]byte("hello"), 0)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestCopyPrefixByBytesNegativeLimit(t *testing.T) {
	t.Parallel()

	got := copyPrefixByBytes([]byte("hello"), -1)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestCopyPrefixByBytesLargerThanData(t *testing.T) {
	t.Parallel()

	data := []byte("hello")
	got := copyPrefixByBytes(data, 100)
	if string(got) != "hello" {
		t.Fatalf("expected full copy, got %q", got)
	}
	// Verify it's a copy, not the same slice.
	got[0] = 'X'
	if data[0] == 'X' {
		t.Fatalf("copyPrefixByBytes should return a copy, not the original")
	}
}

func TestCopyPrefixByBytesExactSize(t *testing.T) {
	t.Parallel()

	data := []byte("abc")
	got := copyPrefixByBytes(data, 3)
	if string(got) != "abc" {
		t.Fatalf("expected full copy, got %q", got)
	}
}

func TestCopyPrefixByBytesMidRune(t *testing.T) {
	t.Parallel()

	// U+1F600 is 4 bytes. Cut at 2 should yield empty after rune correction.
	data := []byte("\U0001F600abc")
	got := copyPrefixByBytes(data, 2)
	if len(got) != 0 {
		t.Fatalf("expected empty after mid-rune cut, got %q (len=%d)", got, len(got))
	}
}

func TestCopyPrefixByBytesASCIICut(t *testing.T) {
	t.Parallel()

	data := []byte("abcdef")
	got := copyPrefixByBytes(data, 3)
	if string(got) != "abc" {
		t.Fatalf("expected 'abc', got %q", got)
	}
}

// --- truncateText tests ---

func TestTruncateTextZeroLimit(t *testing.T) {
	t.Parallel()

	got := truncateText("hello world", 0)
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestTruncateTextSmallLimit(t *testing.T) {
	t.Parallel()

	// Limit smaller than truncation marker.
	got := truncateText("hello world this is a test", 5)
	if len(got) > 5 {
		t.Fatalf("exceeded limit: got len=%d", len(got))
	}
}

func TestTruncateTextNoTruncation(t *testing.T) {
	t.Parallel()

	got := truncateText("hello", 100)
	if got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}
}

func TestTruncateTextMultibyte(t *testing.T) {
	t.Parallel()

	// 10 CJK chars = 30 bytes. Limit to 20.
	s := strings.Repeat("\u4E16", 10)
	got := truncateText(s, 20)
	if len(got) > 20 {
		t.Fatalf("exceeded limit: got len=%d", len(got))
	}
	if !strings.Contains(got, truncationMarker) {
		t.Fatalf("should contain truncation marker, got %q", got)
	}
}

func TestTruncateTextExactFit(t *testing.T) {
	t.Parallel()

	s := "abc"
	got := truncateText(s, 3)
	if got != "abc" {
		t.Fatalf("expected exact fit, got %q", got)
	}
}

// --- BuildUserPrompt edge cases ---

func TestBuildUserPromptEmptyDocsSlice(t *testing.T) {
	t.Parallel()

	got := BuildUserPrompt("Q", "A", []Document{})
	if !strings.Contains(got, "(none)") {
		t.Fatalf("empty docs slice should produce (none), got %q", got)
	}
}

func TestBuildUserPromptDocAutoIDSequential(t *testing.T) {
	t.Parallel()

	docs := []Document{
		{ID: "", Text: "first"},
		{ID: "", Text: "second"},
		{ID: "", Text: "third"},
	}
	got := BuildUserPrompt("Q", "A", docs)
	if !strings.Contains(got, "- [doc-1] first") {
		t.Fatalf("missing doc-1: %q", got)
	}
	if !strings.Contains(got, "- [doc-2] second") {
		t.Fatalf("missing doc-2: %q", got)
	}
	if !strings.Contains(got, "- [doc-3] third") {
		t.Fatalf("missing doc-3: %q", got)
	}
}

func TestBuildUserPromptDocAutoIDSkipsEmpty(t *testing.T) {
	t.Parallel()

	// When doc at index 1 has empty text, doc at index 2 should still be doc-3 (1-based index).
	docs := []Document{
		{ID: "", Text: "first"},
		{ID: "", Text: ""},
		{ID: "", Text: "third"},
	}
	got := BuildUserPrompt("Q", "A", docs)
	if !strings.Contains(got, "- [doc-1] first") {
		t.Fatalf("missing doc-1: %q", got)
	}
	// The empty doc at index 1 is skipped, so no doc-2 appears.
	// The doc at index 2 gets id doc-3 (i+1 where i=2).
	if !strings.Contains(got, "- [doc-3] third") {
		t.Fatalf("expected doc-3 for third doc: %q", got)
	}
}

func TestBuildUserPromptLargeContentTruncation(t *testing.T) {
	t.Parallel()

	// Generate content that will cause prompt-level truncation.
	bigQ := strings.Repeat("q", maxPromptBytes/2)
	bigA := strings.Repeat("a", maxPromptBytes/2)
	bigDoc := strings.Repeat("d", maxDocBytes)
	got := BuildUserPrompt(bigQ, bigA, []Document{
		{ID: "d1", Text: bigDoc},
		{ID: "d2", Text: bigDoc},
	})
	if len(got) > maxPromptBytes {
		t.Fatalf("prompt exceeded max: got=%d max=%d", len(got), maxPromptBytes)
	}
	if !strings.Contains(got, truncationMarker) {
		t.Fatalf("expected truncation marker in output")
	}
}

func TestBuildUserPromptManyDocsLaterTruncated(t *testing.T) {
	t.Parallel()

	// Create many large documents to trigger truncation of later ones.
	docs := make([]Document, 20)
	docText := strings.Repeat("x", maxDocBytes-100)
	for i := range docs {
		docs[i] = Document{ID: "", Text: docText}
	}
	got := BuildUserPrompt("Q", "A", docs)
	if len(got) > maxPromptBytes {
		t.Fatalf("prompt exceeded max: got=%d max=%d", len(got), maxPromptBytes)
	}
	// First doc should be present.
	if !strings.Contains(got, "- [doc-1]") {
		t.Fatalf("first doc should be present")
	}
}

// --- Schema structure tests ---

func TestFaithfulnessSchemaStructure(t *testing.T) {
	t.Parallel()

	var schema map[string]any
	if err := json.Unmarshal(FaithfulnessSchema(), &schema); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("missing required field")
	}
	found := map[string]bool{}
	for _, r := range required {
		found[r.(string)] = true
	}
	if !found["claims"] || !found["reason"] {
		t.Fatalf("schema must require claims and reason, got %v", required)
	}
}

func TestAnswerRelevancySchemaStructure(t *testing.T) {
	t.Parallel()

	var schema map[string]any
	if err := json.Unmarshal(AnswerRelevancySchema(), &schema); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("missing required field")
	}
	found := map[string]bool{}
	for _, r := range required {
		found[r.(string)] = true
	}
	if !found["score"] || !found["reason"] {
		t.Fatalf("schema must require score and reason, got %v", required)
	}
}

func TestContextRelevancySchemaStructure(t *testing.T) {
	t.Parallel()

	var schema map[string]any
	if err := json.Unmarshal(ContextRelevancySchema(), &schema); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("missing required field")
	}
	found := map[string]bool{}
	for _, r := range required {
		found[r.(string)] = true
	}
	if !found["documents"] || !found["reason"] {
		t.Fatalf("schema must require documents and reason, got %v", required)
	}
}

// --- Instruction non-empty tests ---

func TestInstructionsNonEmpty(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		fn   func() string
	}{
		{"Faithfulness", FaithfulnessInstructions},
		{"AnswerRelevancy", AnswerRelevancyInstructions},
		{"ContextRelevancy", ContextRelevancyInstructions},
	} {
		if tc.fn() == "" {
			t.Fatalf("%s instructions must not be empty", tc.name)
		}
	}
}

// --- cappedWriter partial write with truncation ---

func TestCappedWriterPartialWriteThenTruncation(t *testing.T) {
	t.Parallel()

	markerLen := len(truncationMarker)
	limit := markerLen + 5
	var w cappedWriter
	w.limit = limit
	// Write exactly 5 bytes, then write something that exceeds remaining.
	w.writeString("hello")
	w.writeString(strings.Repeat("x", 100))
	if !w.isTruncated() {
		t.Fatalf("should be truncated")
	}
	got := w.String()
	if len(got) > limit {
		t.Fatalf("output exceeded limit: got=%d limit=%d", len(got), limit)
	}
	if !strings.HasSuffix(got, truncationMarker) {
		t.Fatalf("should end with truncation marker, got %q", got)
	}
	if !strings.HasPrefix(got, "hello") {
		t.Fatalf("should start with 'hello', got %q", got)
	}
}

func TestCappedWriterMultipleWritesThenOverflow(t *testing.T) {
	t.Parallel()

	var w cappedWriter
	w.limit = 50
	w.writeString("aaa")
	w.writeString("bbb")
	w.writeString(strings.Repeat("c", 100))
	if !w.isTruncated() {
		t.Fatalf("should be truncated")
	}
	got := w.String()
	if len(got) > 50 {
		t.Fatalf("exceeded limit: len=%d", len(got))
	}
	if !strings.HasPrefix(got, "aaabbb") {
		t.Fatalf("prefix should be preserved, got %q", got)
	}
}
