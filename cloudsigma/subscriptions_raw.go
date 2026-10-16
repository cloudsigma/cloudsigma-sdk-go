package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SubscriptionsRaw is one complete subscription listing envelope. Raw values
// preserve decimal strings, numeric lexemes, unknown fields and null versus
// absent values without float conversion. Descendants and discounts are opaque;
// preservation does not interpret their schema, chains or billing semantics.
// Each call returns independent data.
type SubscriptionsRaw map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (s *SubscriptionsRaw) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("subscriptions response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*s = fields
	return nil
}

// ListRaw reads one GET /subscriptions/ response using the same documented
// status/resource filters and validation as ListWithOptions. Nil or empty opts
// omits filters. The complete envelope, including meta and price, is preserved;
// Response.Meta is not populated. No paging parameters or automatic traversal
// are added. List, ListWithOptions and the shared Create model are unchanged.
// Existing Client.Do decoding/error rules apply; raw access is not schema
// validation or a guarantee of server support for arbitrary nested values.
// Docs: https://docs.cloudsigma.com/en/latest/subscriptions.html#listing
func (s *SubscriptionsService) ListRaw(ctx context.Context, opts *SubscriptionListOptions) (*SubscriptionsRaw, *Response, error) {
	path, err := subscriptionListPath(opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(SubscriptionsRaw)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
