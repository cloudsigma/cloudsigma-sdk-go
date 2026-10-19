package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"
)

// GlobalContextService provides read access to account-wide global context.
// Docs: https://docs.cloudsigma.com/en/latest/server_context.html
type GlobalContextService service

// GlobalContext preserves the account-wide context object as raw JSON members.
// Numeric lexemes, unknown fields and absent-versus-null values are retained.
// Raw preservation does not validate metadata or authorize any context update.
// Each successful call returns independent data.
type GlobalContext map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *GlobalContext) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("global_context response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// GetRaw reads GET /global_context/ without a request body or query parameters.
// The account context is returned directly, not as a collection or server context.
// This method does not read a guest serial device or update any server metadata.
// Existing Client.Do decoding/error rules apply.
func (s *GlobalContextService) GetRaw(ctx context.Context) (*GlobalContext, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "global_context/", nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(GlobalContext)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}

// Update posts an account-wide context object and returns the resulting context.
// The request is a direct object, not a collection or a nested meta attribute.
// It deliberately accepts string values only, without coercion. Nil is refused
// as SDK policy; an empty nonnil map is sent as {}. No automatic GET/merge is
// performed and no deletion, merge or atomicity guarantee is inferred from the
// public example. Include all pairs you intend to retain. Callers must not mutate
// values concurrently. Invalid UTF-8 is refused rather than silently repaired.
// Docs: https://docs.cloudsigma.com/en/latest/server_context.html#global-context
func (s *GlobalContextService) Update(ctx context.Context, values map[string]string) (*GlobalContext, *Response, error) {
	if values == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}
	for k, v := range values {
		if !utf8.ValidString(k) || !utf8.ValidString(v) {
			return nil, nil, fmt.Errorf("invalid global context UTF-8")
		}
	}
	req, err := s.client.NewRequest(http.MethodPost, "global_context/", values)
	if err != nil {
		return nil, nil, err
	}
	value := new(GlobalContext)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
