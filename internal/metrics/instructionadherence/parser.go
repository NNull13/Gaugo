package instructionadherence

import (
	"github.com/nnull13/gaugo/internal/metrics"
)

const parseErrorPrefix = "parse instruction adherence response"

type Instruction struct {
	Text     string `json:"text"`
	Followed bool   `json:"followed"`
	Reason   string `json:"reason"`
}

type Output struct {
	Instructions []Instruction `json:"instructions"`
	Reason       string        `json:"reason"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		Instructions *[]struct {
			Text     *string `json:"text"`
			Followed *bool   `json:"followed"`
			Reason   *string `json:"reason"`
		} `json:"instructions"`
		Reason *string `json:"reason"`
	}
	err := metrics.DecodeStrict(raw, &decoded, parseErrorPrefix)
	if err != nil {
		return Output{}, err
	}
	if decoded.Instructions == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "instructions")
	}
	if decoded.Reason == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "reason")
	}

	instructions := make([]Instruction, len(*decoded.Instructions))
	for i, instruction := range *decoded.Instructions {
		if instruction.Text == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "instruction", i, "text")
		}
		if instruction.Followed == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "instruction", i, "followed")
		}
		if instruction.Reason == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "instruction", i, "reason")
		}
		instructions[i] = Instruction{
			Text:     *instruction.Text,
			Followed: *instruction.Followed,
			Reason:   *instruction.Reason,
		}
	}
	return Output{Instructions: instructions, Reason: *decoded.Reason}, nil
}

func Score(output Output) float64 {
	if len(output.Instructions) == 0 {
		return 0
	}
	followed := 0
	for _, instruction := range output.Instructions {
		if instruction.Followed {
			followed++
		}
	}
	return float64(followed) / float64(len(output.Instructions))
}
