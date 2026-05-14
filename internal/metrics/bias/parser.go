package bias

import (
	"github.com/nnull13/gaugo/internal/metrics"
)

const parseErrorPrefix = "parse bias response"

type Instance struct {
	Text     string `json:"text"`
	BiasType string `json:"bias_type"`
	Reason   string `json:"reason"`
}

type Output struct {
	Biased    bool       `json:"biased"`
	Instances []Instance `json:"instances"`
	Score     float64    `json:"score"`
	Reason    string     `json:"reason"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		Biased    *bool `json:"biased"`
		Instances *[]struct {
			Text     *string `json:"text"`
			BiasType *string `json:"bias_type"`
			Reason   *string `json:"reason"`
		} `json:"instances"`
		Score  *float64 `json:"score"`
		Reason *string  `json:"reason"`
	}
	err := metrics.DecodeStrict(raw, &decoded, parseErrorPrefix)
	if err != nil {
		return Output{}, err
	}
	if decoded.Biased == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "biased")
	}
	if decoded.Instances == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "instances")
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

	instances := make([]Instance, len(*decoded.Instances))
	for i, instance := range *decoded.Instances {
		if instance.Text == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "instance", i, "text")
		}
		if instance.BiasType == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "instance", i, "bias_type")
		}
		if instance.Reason == nil {
			return Output{}, metrics.MissingRequiredNestedField(parseErrorPrefix, "instance", i, "reason")
		}
		instances[i] = Instance{
			Text:     *instance.Text,
			BiasType: *instance.BiasType,
			Reason:   *instance.Reason,
		}
	}
	return Output{
		Biased:    *decoded.Biased,
		Instances: instances,
		Score:     *decoded.Score,
		Reason:    *decoded.Reason,
	}, nil
}

func Score(out Output) float64 {
	return out.Score
}
