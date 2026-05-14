package toxicity

import (
	"github.com/nnull13/gaugo/internal/metrics"
)

const parseErrorPrefix = "parse toxicity response"

type Category struct {
	Name     string  `json:"name"`
	Detected bool    `json:"detected"`
	Severity float64 `json:"severity"`
}

type Output struct {
	Toxic      bool       `json:"toxic"`
	Categories []Category `json:"categories"`
	Score      float64    `json:"score"`
	Reason     string     `json:"reason"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		Toxic      *bool `json:"toxic"`
		Categories *[]struct {
			Name     *string  `json:"name"`
			Detected *bool    `json:"detected"`
			Severity *float64 `json:"severity"`
		} `json:"categories"`
		Score  *float64 `json:"score"`
		Reason *string  `json:"reason"`
	}
	err := metrics.DecodeStrict(raw, &decoded, parseErrorPrefix)
	if err != nil {
		return Output{}, err
	}
	if decoded.Toxic == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "toxic")
	}
	if decoded.Categories == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "categories")
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

	categories := make([]Category, len(*decoded.Categories))
	for i, c := range *decoded.Categories {
		if c.Name == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "category", i, "name")
		}
		if c.Detected == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "category", i, "detected")
		}
		if c.Severity == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "category", i, "severity")
		}
		err = metrics.NestedRange01(parseErrorPrefix, "category", i, "severity", *c.Severity)
		if err != nil {
			return Output{}, err
		}
		categories[i] = Category{
			Name:     *c.Name,
			Detected: *c.Detected,
			Severity: *c.Severity,
		}
	}
	return Output{
		Toxic:      *decoded.Toxic,
		Categories: categories,
		Score:      *decoded.Score,
		Reason:     *decoded.Reason,
	}, nil
}

func Score(out Output) float64 {
	return out.Score
}
