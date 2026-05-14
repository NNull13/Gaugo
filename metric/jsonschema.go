package metric

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

const (
	jsonTypeNull    = "null"
	jsonTypeObject  = "object"
	jsonTypeArray   = "array"
	jsonTypeString  = "string"
	jsonTypeBool    = "boolean"
	jsonTypeNumber  = "number"
	jsonTypeInteger = "integer"
)

// decodeJSONValue decodes a single JSON value from raw, optionally preserving
// numeric precision via json.Number. It rejects trailing tokens.
func decodeJSONValue(raw string, useNumber bool) (any, error) {
	dec := json.NewDecoder(strings.NewReader(raw))
	if useNumber {
		dec.UseNumber()
	}
	var value any
	if err := dec.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	err := dec.Decode(&trailing)
	if err == nil {
		return nil, optionErrorf("unexpected trailing JSON value")
	}
	if err != io.EOF {
		return nil, err
	}
	return value, nil
}

type schemaShape struct {
	Type                 any                        `json:"type"`
	Required             []string                   `json:"required"`
	Properties           map[string]json.RawMessage `json:"properties"`
	Items                json.RawMessage            `json:"items"`
	AdditionalProperties *bool                      `json:"additionalProperties"`
}

// validateSchemaValue performs a minimal JSON-Schema-style validation
// supporting type (string or array), required, properties, items and
// additionalProperties.
func validateSchemaValue(value any, raw json.RawMessage, path string) error {
	var schema schemaShape
	if err := json.Unmarshal(raw, &schema); err != nil {
		return fmt.Errorf("schema at %s is invalid: %w", path, err)
	}

	types, err := schemaTypeList(schema.Type)
	if err != nil {
		return fmt.Errorf("schema at %s is invalid: %w", path, err)
	}
	if len(types) > 0 {
		matched := false
		for _, typ := range types {
			if matchesJSONType(value, typ) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%s expected type %s, got %s", path, strings.Join(types, "|"), jsonTypeName(value))
		}
	}

	if len(schema.Properties) > 0 || len(schema.Required) > 0 || schema.AdditionalProperties != nil {
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s expected object, got %s", path, jsonTypeName(value))
		}
		for _, name := range schema.Required {
			if _, ok := obj[name]; !ok {
				return fmt.Errorf("%s missing required field %q", path, name)
			}
		}
		for name, propertySchema := range schema.Properties {
			child, ok := obj[name]
			if !ok {
				continue
			}
			if err := validateSchemaValue(child, propertySchema, path+"."+name); err != nil {
				return err
			}
		}
		if schema.AdditionalProperties != nil && !*schema.AdditionalProperties {
			for name := range obj {
				if _, ok := schema.Properties[name]; !ok {
					return fmt.Errorf("%s unexpected field %q", path, name)
				}
			}
		}
	}

	if len(schema.Items) > 0 {
		items, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s expected array, got %s", path, jsonTypeName(value))
		}
		for i, item := range items {
			if err := validateSchemaValue(item, schema.Items, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}

	return nil
}

func schemaTypeList(value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	switch typed := value.(type) {
	case string:
		return []string{typed}, nil
	case []any:
		out := make([]string, len(typed))
		for i, item := range typed {
			s, ok := item.(string)
			if !ok {
				return nil, errors.New("type array must contain strings")
			}
			out[i] = s
		}
		return out, nil
	default:
		return nil, errors.New("type must be a string or array of strings")
	}
}

func matchesJSONType(value any, typ string) bool {
	typ = strings.TrimSpace(typ)
	if typ == jsonTypeNull {
		return value == nil
	}
	if value == nil {
		return false
	}
	switch typ {
	case jsonTypeObject:
		_, ok := value.(map[string]any)
		return ok
	case jsonTypeArray:
		_, ok := value.([]any)
		return ok
	case jsonTypeString:
		_, ok := value.(string)
		return ok
	case jsonTypeBool:
		_, ok := value.(bool)
		return ok
	case jsonTypeNumber:
		return isJSONNumber(value)
	case jsonTypeInteger:
		return jsonNumberIsInteger(value)
	default:
		return false
	}
}

func jsonTypeName(value any) string {
	if value == nil {
		return jsonTypeNull
	}
	if isJSONNumber(value) {
		return jsonTypeNumber
	}
	switch reflect.TypeOf(value).Kind() {
	case reflect.Map:
		return jsonTypeObject
	case reflect.Slice:
		return jsonTypeArray
	case reflect.String:
		return jsonTypeString
	case reflect.Bool:
		return jsonTypeBool
	default:
		return reflect.TypeOf(value).String()
	}
}

func isJSONNumber(value any) bool {
	_, ok := jsonNumberRat(value)
	return ok
}

func jsonNumberIsInteger(value any) bool {
	n, ok := jsonNumberRat(value)
	return ok && n.IsInt()
}

func jsonNumbersEqual(left, right any) bool {
	leftNum, ok := jsonNumberRat(left)
	if !ok {
		return false
	}
	rightNum, ok := jsonNumberRat(right)
	if !ok {
		return false
	}
	return leftNum.Cmp(rightNum) == 0
}

func jsonNumberRat(value any) (*big.Rat, bool) {
	switch typed := value.(type) {
	case json.Number:
		return parseJSONNumberRat(typed.String())
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return nil, false
		}
		return parseJSONNumberRat(strconv.FormatFloat(typed, 'g', -1, 64))
	default:
		return nil, false
	}
}

func parseJSONNumberRat(raw string) (*big.Rat, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}

	sign := 1
	if strings.HasPrefix(raw, "-") {
		sign = -1
		raw = raw[1:]
	}
	if raw == "" {
		return nil, false
	}

	mantissa := raw
	exponent := 0
	if before, after, ok := strings.Cut(raw, "e"); ok {
		mantissa = before
		parsed, err := strconv.Atoi(after)
		if err != nil {
			return nil, false
		}
		exponent = parsed
	} else if before, after, ok := strings.Cut(raw, "E"); ok {
		mantissa = before
		parsed, err := strconv.Atoi(after)
		if err != nil {
			return nil, false
		}
		exponent = parsed
	}

	intPart := mantissa
	fracPart := ""
	if before, after, ok := strings.Cut(mantissa, "."); ok {
		intPart = before
		fracPart = after
	}
	if intPart == "" || strings.ContainsAny(fracPart, ".eE") {
		return nil, false
	}

	digits := intPart + fracPart
	if digits == "" {
		return nil, false
	}
	numerator := new(big.Int)
	if _, ok := numerator.SetString(digits, 10); !ok {
		return nil, false
	}
	if sign < 0 {
		numerator.Neg(numerator)
	}

	scale := len(fracPart) - exponent
	rat := new(big.Rat).SetInt(numerator)
	if scale > 0 {
		denominator := pow10(scale)
		rat.Quo(rat, new(big.Rat).SetInt(denominator))
	} else if scale < 0 {
		multiplier := pow10(-scale)
		rat.Mul(rat, new(big.Rat).SetInt(multiplier))
	}
	return rat, true
}

func pow10(exp int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil)
}

func jsonValuesEqual(left, right any) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	if isJSONNumber(left) || isJSONNumber(right) {
		return jsonNumbersEqual(left, right)
	}

	switch l := left.(type) {
	case map[string]any:
		r, ok := right.(map[string]any)
		if !ok || len(l) != len(r) {
			return false
		}
		for key, leftValue := range l {
			rightValue, ok := r[key]
			if !ok || !jsonValuesEqual(leftValue, rightValue) {
				return false
			}
		}
		return true
	case []any:
		r, ok := right.([]any)
		if !ok || len(l) != len(r) {
			return false
		}
		for i, leftValue := range l {
			if !jsonValuesEqual(leftValue, r[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(left, right)
	}
}

// normalizeJSONComparable round-trips a Go value through JSON to ensure
// equality comparisons with decoded JSON values use the canonical form.
func normalizeJSONComparable(value any) (any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return decodeJSONValue(string(raw), true)
}

// lookupJSONPath resolves a dotted path against a decoded JSON value, walking
// map keys and slice indexes.
func lookupJSONPath(value any, path string) (any, bool) {
	current := value
	for part := range strings.SplitSeq(path, ".") {
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[part]
			if !ok {
				return nil, false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}
	return current, true
}
