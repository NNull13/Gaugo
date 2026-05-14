package metric

import (
	"context"
	"encoding/json"
	"time"
)

// Document is one retrieved context record for a case.
type Document struct {
	ID   string
	Text string
}

// Doc is a convenience constructor for Document.
func Doc(id, text string) Document {
	return Document{ID: id, Text: text}
}

// Input is the test input passed to a target system under evaluation.
type Input struct {
	Question string
	Context  []Document
}

// Output is the target system answer under evaluation.
type Output struct {
	Answer string
}

// Expected holds simple non-LLM assertions for a case.
type Expected struct {
	Contains     []string
	Answer       string
	Instructions string
}

// EvalInput is provided to a Metric after a case run completes.
type EvalInput struct {
	CaseName string
	Input    Input
	Output   Output
	Expected Expected
	Elapsed  time.Duration
}

// Judge evaluates metric prompts and must return strictly-structured JSON.
type Judge interface {
	EvaluateJSON(ctx context.Context, req JudgeRequest) (JudgeResponse, error)
}

// JudgeRequest describes a metric evaluation request sent to a Judge.
type JudgeRequest struct {
	Metric               string
	Question             string
	Answer               string
	ExpectedAnswer       string
	ExpectedInstructions string
	ContextDocs          []Document
	Instructions         string
	Schema               json.RawMessage
}

// JudgeResponse contains the raw structured output from a Judge.
type JudgeResponse struct {
	RawJSON   []byte
	Provider  string
	Model     string
	RequestID string
	Latency   time.Duration
}
