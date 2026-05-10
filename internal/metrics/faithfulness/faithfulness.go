package faithfulness

import (
	"fmt"

	"github.com/nnull13/gaugo/internal/jsonx"
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
	if err := jsonx.DecodeStrict(raw, &decoded); err != nil {
		return Output{}, fmt.Errorf("%s: %w", parseErrorPrefix, err)
	}
	if decoded.Claims == nil {
		return Output{}, fmt.Errorf("%s: missing required field claims", parseErrorPrefix)
	}
	if decoded.Reason == nil {
		return Output{}, fmt.Errorf("%s: missing required field reason", parseErrorPrefix)
	}

	claims := make([]Claim, len(*decoded.Claims))
	for i, c := range *decoded.Claims {
		if c.Text == nil {
			return Output{}, fmt.Errorf("%s: claim %d missing required field text", parseErrorPrefix, i)
		}
		if c.Supported == nil {
			return Output{}, fmt.Errorf("%s: claim %d missing required field supported", parseErrorPrefix, i)
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
