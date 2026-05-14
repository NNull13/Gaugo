# Structured Output Metrics

Three deterministic metrics for validating JSON answers. No judge required, no
network calls.

## JSONValidity

Checks whether the answer is valid JSON.

```go
gaugo.JSONValidity()
```

**Required input:** `Answer`.

**Scoring:** `1` if `json.Valid()` passes, `0` otherwise.

**Details:** `{"valid": true}` or `{"valid": false}`.

## SchemaCompliance

Validates the answer against a JSON Schema.

```go
gaugo.SchemaCompliance(gaugo.WithSchema(json.RawMessage(`{
    "type": "object",
    "required": ["status"],
    "properties": {
        "status": { "type": "string" }
    }
}`)))
```

**Required input:** `Answer`.

**Required option:** `WithSchema(json.RawMessage)`. Fails at evaluation time if
the schema is missing or invalid.

**Scoring:** `1` if the answer validates against the schema, `0` otherwise.

**Supported JSON Schema subset:**

| Keyword | Support |
| --- | --- |
| `type` | `string`, `number`, `integer`, `boolean`, `object`, `array`, `null` |
| `properties` | Object property validation |
| `required` | Required field enforcement |
| `items` | Array item schema |
| `additionalProperties` | When `false`, rejects unknown keys |

Numeric values preserve precision through `json.Number`. Nested objects and
arrays are validated recursively.

**Details:** `{"valid": true}` or `{"valid": false, "error": "missing required field: status"}`.

## ExpectedJSON

Checks specific field values in a JSON answer using dot-path lookup.

```go
gaugo.ExpectedJSON(gaugo.WithExpectedFields(map[string]any{
    "status":       "ok",
    "user.name":    "Alice",
    "items.0.id":   42,
}))
```

**Required input:** `Answer`.

**Required option:** `WithExpectedFields(map[string]any)`. Fails at evaluation
time if the field map is missing or empty.

**Scoring:** `matched_fields / total_expected_fields`. All fields must match to
score `1`.

**Path syntax:**

| Path | Resolves to |
| --- | --- |
| `status` | Root key `status` |
| `user.name` | Nested key `name` inside `user` |
| `items.0.id` | Key `id` in first element of array `items` |

**Details structure:**

```json
{
  "fields": [
    { "path": "status", "found": true, "match": true, "expected": "ok", "actual": "ok" },
    { "path": "user.name", "found": true, "match": false, "expected": "Alice", "actual": "Bob" }
  ]
}
```

## Evaluation order

When combining all three, the natural evaluation order is:

1. `JSONValidity` — is it valid JSON at all?
2. `SchemaCompliance` — does it match the expected structure?
3. `ExpectedJSON` — do specific fields have the right values?

All three run independently. A `JSONValidity` failure does not short-circuit
`SchemaCompliance` or `ExpectedJSON`; each produces its own `MetricResult`.

## Example

```go
suite.Case("api response",
    gaugo.Question("Get user status"),
)

suite.Assert(ctx, yourFunc,
    gaugo.JSONValidity(),
    gaugo.SchemaCompliance(gaugo.WithSchema(json.RawMessage(`{
        "type": "object",
        "required": ["status", "user"],
        "properties": {
            "status": { "type": "string" },
            "user": {
                "type": "object",
                "required": ["id", "name"],
                "properties": {
                    "id":   { "type": "integer" },
                    "name": { "type": "string" }
                }
            }
        }
    }`))),
    gaugo.ExpectedJSON(gaugo.WithExpectedFields(map[string]any{
        "status":  "active",
        "user.id": 1,
    })),
)
```
