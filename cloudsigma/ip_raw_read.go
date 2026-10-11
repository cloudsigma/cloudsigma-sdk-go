package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// GetRaw retrieves a single IP as an owned JSON object, preserving every
// field's raw JSON representation, including null, empty lists and unknown
// nested values. It does not alter IP or its shared update request shape.
// This is response representation, NOT writable-field authorization. Duplicate
// keys use encoding/json's last-value behavior; object ordering is not retained.
// Only canonical dotted IPv4 is accepted as bounded SDK policy matching the
// public address example despite its UUID placeholder. IPv6 route semantics
// remain unresolved here; this does not deny platform IPv6 support. Null first
// values are refused with Response retained; decode failures return nil Response.
// Only one JSON value is decoded: trailing documents/junk are not validated.
// No response link traversal, polling or followup requests occur.
// Docs: https://docs.cloudsigma.com/en/latest/networking.html#get-single-ip
func (s *IPsService) GetRaw(ctx context.Context, id string) (map[string]json.RawMessage, *Response, error) {
	if id == "" {
		return nil, nil, ErrEmptyArgument
	}
	parsed := net.ParseIP(id)
	if parsed == nil || parsed.To4() == nil || strings.Contains(id, ":") || parsed.String() != id {
		return nil, nil, fmt.Errorf("raw IP read requires canonical dotted IPv4")
	}
	q, err := s.client.NewRequest(http.MethodGet, "ips/"+id+"/", nil)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	r, err := s.client.Do(ctx, q, &out)
	if err != nil {
		return nil, r, err
	}
	if out == nil {
		return nil, r, fmt.Errorf("expected IP response object")
	}
	return out, r, nil
}
