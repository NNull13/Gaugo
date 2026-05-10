package gaugo

import (
	"context"
	"encoding/json"
	"time"
)

// Judge evaluates metric prompts and must return strictly-structured JSON.
type Judge interface {
	EvaluateJSON(ctx context.Context, req JudgeRequest) (JudgeResponse, error)
}

// JudgeRequest describes a metric evaluation request sent to a Judge.
type JudgeRequest struct {
	Metric       string
	Question     string
	Answer       string
	ContextDocs  []Document
	Instructions string
	Schema       json.RawMessage
}

// JudgeResponse contains the raw structured output from a Judge.
type JudgeResponse struct {
	RawJSON  []byte
	Provider string
	Model    string
	Latency  time.Duration
}
