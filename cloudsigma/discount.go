package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// DiscountService provides the public discount read operation.
// Docs: https://docs.cloudsigma.com/en/latest/billing.html
// It does not expose mutations, detail routes or automatic traversal.
type DiscountService service

// Discount is a raw JSON object preserving decimal strings, numeric lexemes,
// unknown fields and absent-versus-null values without float conversion.
// The complete collection envelope is retained, including the documented period
// and decimal-string value fields. No discount arithmetic or schema validation
// is performed. Each call returns independent data.
type Discount map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *Discount) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("discount response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// ListRaw reads one GET /discount/ collection envelope. It preserves meta,
// objects and unknown envelope fields without projecting them into Response.Meta.
// No pagination traversal, discount calculation, billing interpretation or
// schema validation is performed. Nil opts omits paging parameters; a nonnil
// zero-value opts requests limit=0. No date or other filters are added.
// Existing Client.Do decoding/error rules apply.
func (s *DiscountService) ListRaw(ctx context.Context, opts *ListOptions) (*Discount, *Response, error) {
	path, err := addOptions("discount/", opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(Discount)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
