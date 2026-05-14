package citationaccuracy

import (
	"github.com/nnull13/gaugo/internal/metrics"
)

const parseErrorPrefix = "parse citation accuracy response"

type Citation struct {
	Text     string `json:"text"`
	DocID    string `json:"doc_id"`
	Accurate bool   `json:"accurate"`
	Reason   string `json:"reason"`
}

type Output struct {
	Citations []Citation `json:"citations"`
	Reason    string     `json:"reason"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		Citations *[]struct {
			Text     *string `json:"text"`
			DocID    *string `json:"doc_id"`
			Accurate *bool   `json:"accurate"`
			Reason   *string `json:"reason"`
		} `json:"citations"`
		Reason *string `json:"reason"`
	}
	err := metrics.DecodeStrict(raw, &decoded, parseErrorPrefix)
	if err != nil {
		return Output{}, err
	}
	if decoded.Citations == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "citations")
	}
	if decoded.Reason == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "reason")
	}

	citations := make([]Citation, len(*decoded.Citations))
	for i, c := range *decoded.Citations {
		if c.Text == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "citation", i, "text")
		}
		if c.DocID == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "citation", i, "doc_id")
		}
		if c.Accurate == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "citation", i, "accurate")
		}
		if c.Reason == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "citation", i, "reason")
		}
		citations[i] = Citation{
			Text:     *c.Text,
			DocID:    *c.DocID,
			Accurate: *c.Accurate,
			Reason:   *c.Reason,
		}
	}
	return Output{Citations: citations, Reason: *decoded.Reason}, nil
}

func Score(citations []Citation) float64 {
	if len(citations) == 0 {
		return 0
	}
	accurate := 0
	for _, c := range citations {
		if c.Accurate {
			accurate++
		}
	}
	return float64(accurate) / float64(len(citations))
}
