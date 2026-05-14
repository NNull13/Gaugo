// Package metrics provides validation helpers shared by every built-in
// judge-metric sub-package under internal/metrics/*. The sub-packages follow
// a uniform shape that callers in the root gaugo package can compose
// generically:
//
//  1. Parse(raw []byte) (Output, error) — strict-decode the judge's JSON
//     response, returning a typed Output with all required fields validated.
//  2. Score(...) float64 — optional aggregation when the metric score is
//     derived from sub-fields (e.g. claim ratios, weighted averages).
//
// The helpers in this file (DecodeStrict, Range01, MissingRequiredField, …)
// are deliberately small and side-effect free so they can be reused by future
// user-defined metrics. The sub-packages remain unexported in v1.x; a future
// release may promote them to a public extension surface.
package metrics

import (
	"fmt"

	"github.com/nnull13/gaugo/internal/strictjson"
)

const (
	errFormatPrefixWrap                 = "%s: %w"
	errFormatMissingRequiredField       = "%s: missing required field %s"
	errFormatMissingRequiredNestedField = "%s: %s %d missing required field %s"
	errFormatRange01                    = "%s: %s must be in [0,1], got %f"
	errFormatNestedRange01              = "%s: %s %d %s must be in [0,1], got %f"
)

func DecodeStrict(raw []byte, v any, prefix string) error {
	err := strictjson.DecodeStrict(raw, v)
	if err != nil {
		return fmt.Errorf(errFormatPrefixWrap, prefix, err)
	}
	return nil
}

func MissingRequiredField(prefix, field string) error {
	return fmt.Errorf(errFormatMissingRequiredField, prefix, field)
}

func MissingRequiredNestedField(prefix, kind string, index int, field string) error {
	return fmt.Errorf(errFormatMissingRequiredNestedField, prefix, kind, index, field)
}

func Range01(prefix, field string, value float64) error {
	if value < 0 || value > 1 {
		return fmt.Errorf(errFormatRange01, prefix, field, value)
	}
	return nil
}

func NestedRange01(prefix, kind string, index int, field string, value float64) error {
	if value < 0 || value > 1 {
		return fmt.Errorf(errFormatNestedRange01, prefix, kind, index, field, value)
	}
	return nil
}
