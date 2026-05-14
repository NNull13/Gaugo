package answerrelevancy

import (
	"github.com/nnull13/gaugo/internal/metrics"
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
	err := metrics.DecodeStrict(raw, &decoded, parseErrorPrefix)
	if err != nil {
		return Output{}, err
	}
	if decoded.Score == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "score")
	}
	if decoded.Reason == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "reason")
	}
	err = metrics.Range01(parseErrorPrefix, "score", *decoded.Score)
	if err != nil {
		return Output{}, err
	}
	return Output{
		Score:  *decoded.Score,
		Reason: *decoded.Reason,
		Issues: decoded.Issues,
	}, nil
}
