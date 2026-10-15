package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// KeypairRaw preserves a keypair object or collection envelope, including private
// key material. It is NOT redacted. Unknown fields, numeric lexemes and absent
// versus null fields survive. It validates only the outer JSON object, not keys
// or nested schemas. Ordinary copies may alias; separate reads own fresh data.
type KeypairRaw map[string]json.RawMessage

// UnmarshalJSON replaces the receiver only after decoding a JSON object.
func (k *KeypairRaw) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("keypair response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*k = fields
	return nil
}

// GetRaw reads GET /keypairs/{uuid}/ without changing authentication or OTP
// handling. Prefer Pubkeys.GetRaw when private material is not needed. The new
// method accepts only ASCII letters, digits and hyphens as SDK route policy.
// Existing typed reads and shared mutation serialization remain unchanged.
// Docs: https://docs.cloudsigma.com/en/latest/keypairs.html
func (s *KeypairsService) GetRaw(ctx context.Context, uuid string) (*KeypairRaw, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	for _, ch := range uuid {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return nil, nil, ErrEmptyArgument
		}
	}
	return s.readRaw(ctx, "keypairs/"+uuid+"/")
}

// ListRaw reads one GET /keypairs/ page, preserving the entire objects/meta
// envelope, including unknown metadata. Response.Meta is not populated; inspect
// the raw meta field instead. Nil options omit query; nonnil options include
// limit even when zero. Negative pagination is refused as SDK policy. No pages
// are followed. Existing Client.Do decoding and error behavior applies.
// Docs: https://docs.cloudsigma.com/en/latest/pagination.html
func (s *KeypairsService) ListRaw(ctx context.Context, opts *ListOptions) (*KeypairRaw, *Response, error) {
	if opts != nil && (opts.Limit < 0 || opts.Offset < 0) {
		return nil, nil, fmt.Errorf("negative keypair pagination")
	}
	path, err := addOptions("keypairs/", opts)
	if err != nil {
		return nil, nil, err
	}
	return s.readRaw(ctx, path)
}

func (s *KeypairsService) readRaw(ctx context.Context, path string) (*KeypairRaw, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(KeypairRaw)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
