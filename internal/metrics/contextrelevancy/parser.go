package contextrelevancy

import (
	"github.com/nnull13/gaugo/internal/metrics"
)

const parseErrorPrefix = "parse context relevancy response"

type Document struct {
	ID     string  `json:"id"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

type Output struct {
	Documents []Document `json:"documents"`
	Reason    string     `json:"reason"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		Documents *[]struct {
			ID     *string  `json:"id"`
			Score  *float64 `json:"score"`
			Reason *string  `json:"reason"`
		} `json:"documents"`
		Reason *string `json:"reason"`
	}
	err := metrics.DecodeStrict(raw, &decoded, parseErrorPrefix)
	if err != nil {
		return Output{}, err
	}
	if decoded.Documents == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "documents")
	}
	if decoded.Reason == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "reason")
	}

	documents := make([]Document, len(*decoded.Documents))
	for i, d := range *decoded.Documents {
		if d.ID == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "document", i, "id")
		}
		if d.Score == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "document", i, "score")
		}
		if d.Reason == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "document", i, "reason")
		}
		err = metrics.NestedRange01(parseErrorPrefix, "document", i, "score", *d.Score)
		if err != nil {
			return Output{}, err
		}
		documents[i] = Document{
			ID:     *d.ID,
			Score:  *d.Score,
			Reason: *d.Reason,
		}
	}
	return Output{Documents: documents, Reason: *decoded.Reason}, nil
}

func Score(documents []Document) float64 {
	if len(documents) == 0 {
		return 0
	}
	var total float64
	for _, d := range documents {
		total += d.Score
	}
	return total / float64(len(documents))
}
