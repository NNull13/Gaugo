package hallucination

import (
	"github.com/nnull13/gaugo/internal/metrics"
)

const parseErrorPrefix = "parse hallucination response"

type Claim struct {
	Text         string `json:"text"`
	Hallucinated bool   `json:"hallucinated"`
	Reason       string `json:"reason"`
}

type Output struct {
	Claims []Claim `json:"claims"`
	Reason string  `json:"reason"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		Claims *[]struct {
			Text         *string `json:"text"`
			Hallucinated *bool   `json:"hallucinated"`
			Reason       *string `json:"reason"`
		} `json:"claims"`
		Reason *string `json:"reason"`
	}
	err := metrics.DecodeStrict(raw, &decoded, parseErrorPrefix)
	if err != nil {
		return Output{}, err
	}
	if decoded.Claims == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "claims")
	}
	if decoded.Reason == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "reason")
	}

	claims := make([]Claim, len(*decoded.Claims))
	for i, c := range *decoded.Claims {
		if c.Text == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "claim", i, "text")
		}
		if c.Hallucinated == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "claim", i, "hallucinated")
		}
		if c.Reason == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "claim", i, "reason")
		}
		claims[i] = Claim{
			Text:         *c.Text,
			Hallucinated: *c.Hallucinated,
			Reason:       *c.Reason,
		}
	}
	return Output{Claims: claims, Reason: *decoded.Reason}, nil
}

func Score(claims []Claim) float64 {
	if len(claims) == 0 {
		return 1
	}
	hallucinated := 0
	for _, c := range claims {
		if c.Hallucinated {
			hallucinated++
		}
	}
	return 1 - float64(hallucinated)/float64(len(claims))
}
