package faithfulness

import (
	"github.com/nnull13/gaugo/internal/metrics"
)

const parseErrorPrefix = "parse faithfulness response"

type Claim struct {
	Text      string   `json:"text"`
	Supported bool     `json:"supported"`
	Evidence  []string `json:"evidence,omitempty"`
}

type Output struct {
	Claims []Claim `json:"claims"`
	Reason string  `json:"reason"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		Claims *[]struct {
			Text      *string  `json:"text"`
			Supported *bool    `json:"supported"`
			Evidence  []string `json:"evidence,omitempty"`
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
		if c.Supported == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "claim", i, "supported")
		}
		claims[i] = Claim{
			Text:      *c.Text,
			Supported: *c.Supported,
			Evidence:  c.Evidence,
		}
	}
	return Output{Claims: claims, Reason: *decoded.Reason}, nil
}

func Score(claims []Claim) float64 {
	if len(claims) == 0 {
		return 0
	}
	supported := 0
	for _, c := range claims {
		if c.Supported {
			supported++
		}
	}
	return float64(supported) / float64(len(claims))
}
