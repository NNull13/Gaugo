package completeness

import (
	"github.com/nnull13/gaugo/internal/metrics"
)

const parseErrorPrefix = "parse completeness response"

type Missing struct {
	Aspect string `json:"aspect"`
	Reason string `json:"reason"`
}

type Output struct {
	Score   float64   `json:"score"`
	Missing []Missing `json:"missing"`
	Reason  string    `json:"reason"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		Score   *float64 `json:"score"`
		Missing *[]struct {
			Aspect *string `json:"aspect"`
			Reason *string `json:"reason"`
		} `json:"missing"`
		Reason *string `json:"reason"`
	}
	err := metrics.DecodeStrict(raw, &decoded, parseErrorPrefix)
	if err != nil {
		return Output{}, err
	}
	if decoded.Score == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "score")
	}
	if decoded.Missing == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "missing")
	}
	if decoded.Reason == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "reason")
	}
	err = metrics.Range01(parseErrorPrefix, "score", *decoded.Score)
	if err != nil {
		return Output{}, err
	}

	missing := make([]Missing, len(*decoded.Missing))
	for i, item := range *decoded.Missing {
		if item.Aspect == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "missing item", i, "aspect")
		}
		if item.Reason == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "missing item", i, "reason")
		}
		missing[i] = Missing{
			Aspect: *item.Aspect,
			Reason: *item.Reason,
		}
	}
	return Output{Score: *decoded.Score, Missing: missing, Reason: *decoded.Reason}, nil
}

func Score(output Output) float64 {
	return output.Score
}
