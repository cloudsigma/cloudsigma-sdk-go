package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SnapshotRaw preserves one documented snapshot response without projecting fields.
// Raw values retain unknown fields, numeric lexemes and missing versus null.
// It does not validate snapshot fields or redact an unexpected server field.
type SnapshotRaw map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (c *SnapshotRaw) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("snapshot response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*c = fields
	return nil
}

// GetRaw reads the documented snapshot singleton without query options.
// Existing Get and shared Snapshot mutation serialization remain unchanged.
// Each call owns fresh data; ordinary copies may alias. Existing Do semantics apply.
// Docs: https://docs.cloudsigma.com/en/latest/snapshots.html#list-single-snapshot
func (s *SnapshotsService) GetRaw(ctx context.Context, uuid string) (*SnapshotRaw, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	// Treat the identifier as one path segment, not a query or alternate route.
	for _, ch := range uuid {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return nil, nil, ErrEmptyArgument
		}
	}
	req, err := s.client.NewRequest(http.MethodGet, "snapshots/"+uuid+"/", nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(SnapshotRaw)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
