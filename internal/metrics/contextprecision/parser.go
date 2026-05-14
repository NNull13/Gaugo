package contextprecision

import (
	"github.com/nnull13/gaugo/internal/metrics"
)

const parseErrorPrefix = "parse context precision response"

type Document struct {
	ID     string `json:"id"`
	Useful bool   `json:"useful"`
	Reason string `json:"reason"`
}

type Output struct {
	Documents []Document `json:"documents"`
	Reason    string     `json:"reason"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		Documents *[]struct {
			ID     *string `json:"id"`
			Useful *bool   `json:"useful"`
			Reason *string `json:"reason"`
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
		if d.Useful == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "document", i, "useful")
		}
		if d.Reason == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "document", i, "reason")
		}
		documents[i] = Document{
			ID:     *d.ID,
			Useful: *d.Useful,
			Reason: *d.Reason,
		}
	}
	return Output{Documents: documents, Reason: *decoded.Reason}, nil
}

func Score(documents []Document) float64 {
	if len(documents) == 0 {
		return 0
	}
	useful := 0
	for _, d := range documents {
		if d.Useful {
			useful++
		}
	}
	return float64(useful) / float64(len(documents))
}
