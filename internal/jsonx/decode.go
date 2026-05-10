package jsonx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// DecodeStrict decodes JSON with unknown-field and trailing-token rejection.
func DecodeStrict(data []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}

	var extra any
	if err := dec.Decode(&extra); err != nil {
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("decode trailing json: %w", err)
	}
	return fmt.Errorf("unexpected trailing json value")
}
