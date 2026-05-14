package metric

import (
	"encoding/json"
	"strings"
	"unicode"
)

// boolScore maps a boolean to the canonical pass/fail score domain.
func boolScore(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

// deterministicResult builds a Result for a non-judge metric, marshaling
// details consistently across implementations.
func deterministicResult(name string, score, threshold float64, reason string, details any) (Result, error) {
	raw, err := json.Marshal(details)
	if err != nil {
		return Result{}, newDetailsMarshalError(name, err)
	}
	return Result{
		Name:    name,
		Score:   score,
		Pass:    score >= threshold,
		Reason:  reason,
		Details: raw,
	}, nil
}

// jaccardSimilarity returns the Jaccard index between two strings tokenized on
// non-alphanumeric runes, plus intersection/union sizes.
func jaccardSimilarity(answer, expected string) (float64, int, int) {
	a := tokenSet(answer)
	b := tokenSet(expected)
	union := len(a)
	intersection := 0
	for token := range b {
		if _, ok := a[token]; ok {
			intersection++
		} else {
			union++
		}
	}
	if union == 0 {
		return 0, 0, 0
	}
	return float64(intersection) / float64(union), intersection, union
}

func tokenSet(value string) map[string]struct{} {
	parts := strings.FieldsFunc(strings.ToLower(value), isTokenSeparator)
	out := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		if part != "" {
			out[part] = struct{}{}
		}
	}
	return out
}

func isTokenSeparator(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

// labelOf converts a PascalCase metric name to a lowercase, space-separated
// human-readable label, keeping consecutive uppercase letters together as a
// single acronym. Examples:
//
//	"AnswerRelevancy"      → "answer relevancy"
//	"ExpectedJSON"         → "expected json"
//	"InstructionAdherence" → "instruction adherence"
//	"GEval"                → "g eval"
func labelOf(name string) string {
	if name == "" {
		return ""
	}
	runes := []rune(name)
	var b strings.Builder
	b.Grow(len(name) + 4)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			// Insert a space when this uppercase letter starts a new word: either
			// the previous rune was lowercase, or it was uppercase but the next
			// rune is lowercase (end of an acronym, start of a new word).
			if unicode.IsLower(prev) || (unicode.IsUpper(prev) && nextIsLower) {
				b.WriteByte(' ')
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
