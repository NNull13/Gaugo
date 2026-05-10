package answerrelevancy

import (
	"fmt"

	"github.com/nnull13/gaugo/internal/jsonx"
)

const parseErrorPrefix = "parse answer relevancy response"

type Output struct {
	Score  float64  `json:"score"`
	Reason string   `json:"reason"`
	Issues []string `json:"issues,omitempty"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		Score  *float64 `json:"score"`
		Reason *string  `json:"reason"`
		Issues []string `json:"issues,omitempty"`
	}
	if err := jsonx.DecodeStrict(raw, &decoded); err != nil {
		return Output{}, fmt.Errorf("%s: %w", parseErrorPrefix, err)
	}
	if decoded.Score == nil {
		return Output{}, fmt.Errorf("%s: missing required field score", parseErrorPrefix)
	}
	if decoded.Reason == nil {
		return Output{}, fmt.Errorf("%s: missing required field reason", parseErrorPrefix)
	}
	if *decoded.Score < 0 || *decoded.Score > 1 {
		return Output{}, fmt.Errorf("%s: score must be in [0,1], got %f", parseErrorPrefix, *decoded.Score)
	}
	return Output{
		Score:  *decoded.Score,
		Reason: *decoded.Reason,
		Issues: decoded.Issues,
	}, nil
}
