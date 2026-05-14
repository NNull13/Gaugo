// Package strictjson decodes JSON into Go values while rejecting unknown
// fields and trailing tokens. It powers metric parsers that need to fail
// hard on judge responses that do not match the expected shape.
package strictjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	errDecodeJSON         = "decode json"
	errDecodeTrailingJSON = "decode trailing json"
)

var errUnexpectedTrailingJSONValue = errors.New("unexpected trailing json value")

// DecodeStrict decodes JSON with unknown-field and trailing-token rejection.
func DecodeStrict(data []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	err := dec.Decode(out)
	if err != nil {
		return fmt.Errorf("%s: %w", errDecodeJSON, err)
	}

	var extra any
	err = dec.Decode(&extra)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("%s: %w", errDecodeTrailingJSON, err)
	}
	return errUnexpectedTrailingJSONValue
}
