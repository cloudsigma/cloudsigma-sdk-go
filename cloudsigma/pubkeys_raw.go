package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// PubkeyRaw preserves one documented public SSH-key response without projecting fields.
// Raw values retain unknown fields, numeric lexemes and missing versus null.
// It does not validate key material or redact an unexpected server field.
type PubkeyRaw map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (c *PubkeyRaw) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("pubkey response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*c = fields
	return nil
}

// GetRaw reads the documented public-key singleton without OTP or query options.
// Existing Get and shared Keypair mutation serialization remain unchanged.
// Each call owns fresh data; ordinary copies may alias. Existing Do semantics apply.
// Docs: https://docs.cloudsigma.com/en/latest/keypairs.html#pubkeys-resource
func (s *PubkeysService) GetRaw(ctx context.Context, uuid string) (*PubkeyRaw, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	// Treat the identifier as one path segment, not a query or alternate route.
	for _, ch := range uuid {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return nil, nil, ErrEmptyArgument
		}
	}
	req, err := s.client.NewRequest(http.MethodGet, "pubkeys/"+uuid+"/", nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(PubkeyRaw)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
