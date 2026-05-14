package conciseness

import (
	"github.com/nnull13/gaugo/internal/metrics"
)

const parseErrorPrefix = "parse conciseness response"

type Issue struct {
	Text   string `json:"text"`
	Reason string `json:"reason"`
}

type Output struct {
	Score  float64 `json:"score"`
	Issues []Issue `json:"issues"`
	Reason string  `json:"reason"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		Score  *float64 `json:"score"`
		Issues *[]struct {
			Text   *string `json:"text"`
			Reason *string `json:"reason"`
		} `json:"issues"`
		Reason *string `json:"reason"`
	}
	err := metrics.DecodeStrict(raw, &decoded, parseErrorPrefix)
	if err != nil {
		return Output{}, err
	}
	if decoded.Score == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "score")
	}
	if decoded.Issues == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "issues")
	}
	if decoded.Reason == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "reason")
	}
	err = metrics.Range01(parseErrorPrefix, "score", *decoded.Score)
	if err != nil {
		return Output{}, err
	}

	issues := make([]Issue, len(*decoded.Issues))
	for i, issue := range *decoded.Issues {
		if issue.Text == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "issue", i, "text")
		}
		if issue.Reason == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "issue", i, "reason")
		}
		issues[i] = Issue{
			Text:   *issue.Text,
			Reason: *issue.Reason,
		}
	}
	return Output{Score: *decoded.Score, Issues: issues, Reason: *decoded.Reason}, nil
}

func Score(output Output) float64 {
	return output.Score
}
