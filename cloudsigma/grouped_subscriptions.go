package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GroupedSubscriptionsService provides the public grouped subscriptions read operation.
// Docs: https://docs.cloudsigma.com/en/latest/subscriptions.html
// It does not expose mutations, detail routes or automatic traversal.
type GroupedSubscriptionsService service

// GroupedSubscriptions is a raw JSON object preserving decimal strings, numeric lexemes,
// unknown fields and absent-versus-null values without float conversion.
// Populated descendants are opaque: their representation is not specified by the
// inspected public documentation. The complete collection envelope is retained.
// Raw preservation does not validate chain or billing semantics or imply server
// acceptance. Each call returns independent data.
type GroupedSubscriptions map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *GroupedSubscriptions) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("groupedsubscriptions response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// ListRaw reads one GET /groupedsubscriptions/ collection envelope. It preserves meta,
// objects and unknown envelope fields without projecting them into Response.Meta.
// No pagination traversal, chain calculation, billing interpretation or
// schema validation is performed. Nil opts omits paging parameters; a nonnil
// zero-value opts requests limit=0. No date or other filters are added.
// Existing Client.Do decoding/error rules apply.
func (s *GroupedSubscriptionsService) ListRaw(ctx context.Context, opts *ListOptions) (*GroupedSubscriptions, *Response, error) {
	path, err := addOptions("groupedsubscriptions/", opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(GroupedSubscriptions)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
