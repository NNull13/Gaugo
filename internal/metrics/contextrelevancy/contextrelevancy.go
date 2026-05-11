package contextrelevancy

import (
	"fmt"

	"github.com/nnull13/gaugo/internal/jsonx"
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
	if err := jsonx.DecodeStrict(raw, &decoded); err != nil {
		return Output{}, fmt.Errorf("%s: %w", parseErrorPrefix, err)
	}
	if decoded.Documents == nil {
		return Output{}, fmt.Errorf("%s: missing required field documents", parseErrorPrefix)
	}
	if decoded.Reason == nil {
		return Output{}, fmt.Errorf("%s: missing required field reason", parseErrorPrefix)
	}

	documents := make([]Document, len(*decoded.Documents))
	for i, d := range *decoded.Documents {
		if d.ID == nil {
			return Output{}, fmt.Errorf("%s: document %d missing required field id", parseErrorPrefix, i)
		}
		if d.Score == nil {
			return Output{}, fmt.Errorf("%s: document %d missing required field score", parseErrorPrefix, i)
		}
		if d.Reason == nil {
			return Output{}, fmt.Errorf("%s: document %d missing required field reason", parseErrorPrefix, i)
		}
		if *d.Score < 0 || *d.Score > 1 {
			return Output{}, fmt.Errorf("%s: document %d score must be in [0,1], got %f", parseErrorPrefix, i, *d.Score)
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
