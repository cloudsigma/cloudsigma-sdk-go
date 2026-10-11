# Opt-in full-definition Resize requests

The public [Resize (Update or Fail) documentation](https://cloudsigma-docs.readthedocs.io/en/latest/drives.html#resizing-update-or-fail) requires a full drive definition. `ResizeDefinition` accepts a JSON object without converting it through the legacy `Drive` request model:

```go
var definition cloudsigma.DriveDefinition
if err := json.Unmarshal(fullDefinitionJSON, &definition); err != nil {
    return err
}
drives, response, err := client.Drives.ResizeDefinition(ctx, driveID, definition)
```

Use an explicitly prepared, public-API-supported full definition. This is not advice to echo every response field back to the server. The method preserves unknown keys, explicit `false`, `0`, `[]`, `{}`, `null`, nested values and large JSON numbers using `json.RawMessage`. It may normalize whitespace, escaping and object key order. Standard JSON decoding uses last-value handling for duplicate keys.

Do **not** first decode into `Drive` and marshal it back: unmodeled fields and `omitempty` values can already have been lost. Existing `Resize` and `Drive` behavior is unchanged; opting in is explicit.

Identifiers must be nonempty and contain only ASCII letters, digits, `-` or `_`. This conservative SDK URL-safety policy prevents identifiers changing the resource/action URL. It is not a claim about every identifier accepted by the public API. The documented hyphenated UUID shape is supported.

Nil definitions and malformed raw JSON values fail locally. Empty objects are not rejected by invented required-field rules: callers remain responsible for the complete public API definition. Do not modify the map or its byte slices concurrently with the call.

The returned `[]Drive` uses the existing modeled response decoder; it is **not** a lossless raw response. Numeric license compatibility and existing context/error conventions are retained. HTTP 202 indicates acceptance, not completed resizing; the public example returns `status: "resizing"` and a size different from the requested size. Offline mocked 202 tests prove request/decoder behavior only, not live server acceptance or completion.

Only public API documentation defines server behavior. Raw preservation does not establish tag-clearing, omission, permission or field-writability semantics.
