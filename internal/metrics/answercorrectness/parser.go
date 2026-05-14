package answercorrectness

import (
	"github.com/nnull13/gaugo/internal/metrics"
)

const parseErrorPrefix = "parse answer correctness response"

type Statement struct {
	Text    string `json:"text"`
	Correct bool   `json:"correct"`
	Reason  string `json:"reason"`
}

type Output struct {
	Statements []Statement `json:"statements"`
	Score      float64     `json:"score"`
	Reason     string      `json:"reason"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		Statements *[]struct {
			Text    *string `json:"text"`
			Correct *bool   `json:"correct"`
			Reason  *string `json:"reason"`
		} `json:"statements"`
		Score  *float64 `json:"score"`
		Reason *string  `json:"reason"`
	}
	err := metrics.DecodeStrict(raw, &decoded, parseErrorPrefix)
	if err != nil {
		return Output{}, err
	}
	if decoded.Statements == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "statements")
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

	statements := make([]Statement, len(*decoded.Statements))
	for i, s := range *decoded.Statements {
		if s.Text == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "statement", i, "text")
		}
		if s.Correct == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "statement", i, "correct")
		}
		if s.Reason == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "statement", i, "reason")
		}
		statements[i] = Statement{
			Text:    *s.Text,
			Correct: *s.Correct,
			Reason:  *s.Reason,
		}
	}
	return Output{Statements: statements, Score: *decoded.Score, Reason: *decoded.Reason}, nil
}

func Score(out Output) float64 {
	return out.Score
}
