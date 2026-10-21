package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// TagsPage preserves a dynamic objects/meta envelope without schema validation
// or redaction. Each ListRaw owns fresh raw values; ordinary map copies may alias.
type TagsPage map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *TagsPage) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("tags response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// ListRaw reads one tag collection with its complete raw envelope.
// Nil options omit pagination; a nonnil zero value sends limit=0. Negative
// pagination is rejected locally. No links are followed, and Response.Meta is
// not populated. Values are not schema-validated or redacted. This read does not
// establish tag association changes.
// Docs: https://docs.cloudsigma.com/en/latest/tags.html#listing
func (s *TagsService) ListRaw(ctx context.Context, opts *ListOptions) (*TagsPage, *Response, error) {
	if opts != nil && (opts.Limit < 0 || opts.Offset < 0) {
		return nil, nil, fmt.Errorf("negative tags pagination")
	}
	path, err := addOptions("tags/", opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(TagsPage)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
