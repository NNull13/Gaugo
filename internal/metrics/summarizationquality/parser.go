package summarizationquality

import (
	"github.com/nnull13/gaugo/internal/metrics"
)

const parseErrorPrefix = "parse summarization quality response"

type Output struct {
	CoverageScore    float64 `json:"coverage_score"`
	FidelityScore    float64 `json:"fidelity_score"`
	ConcisenessScore float64 `json:"conciseness_score"`
	Reason           string  `json:"reason"`
}

func Parse(raw []byte) (Output, error) {
	var decoded struct {
		CoverageScore    *float64 `json:"coverage_score"`
		FidelityScore    *float64 `json:"fidelity_score"`
		ConcisenessScore *float64 `json:"conciseness_score"`
		Reason           *string  `json:"reason"`
	}
	err := metrics.DecodeStrict(raw, &decoded, parseErrorPrefix)
	if err != nil {
		return Output{}, err
	}
	if decoded.CoverageScore == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "coverage_score")
	}
	if decoded.FidelityScore == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "fidelity_score")
	}
	if decoded.ConcisenessScore == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "conciseness_score")
	}
	if decoded.Reason == nil {
		return Output{}, metrics.MissingRequiredField(parseErrorPrefix, "reason")
	}
	err = metrics.Range01(parseErrorPrefix, "coverage_score", *decoded.CoverageScore)
	if err != nil {
		return Output{}, err
	}
	err = metrics.Range01(parseErrorPrefix, "fidelity_score", *decoded.FidelityScore)
	if err != nil {
		return Output{}, err
	}
	err = metrics.Range01(parseErrorPrefix, "conciseness_score", *decoded.ConcisenessScore)
	if err != nil {
		return Output{}, err
	}
	return Output{
		CoverageScore:    *decoded.CoverageScore,
		FidelityScore:    *decoded.FidelityScore,
		ConcisenessScore: *decoded.ConcisenessScore,
		Reason:           *decoded.Reason,
	}, nil
}

func Score(coverageScore, fidelityScore, concisenessScore float64) float64 {
	return (coverageScore + fidelityScore + concisenessScore) / 3
}
