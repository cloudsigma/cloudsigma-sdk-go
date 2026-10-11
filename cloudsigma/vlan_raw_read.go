package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetRaw retrieves a single VLAN as an owned JSON object, preserving every
// field's raw JSON representation, including null, empty lists and unknown
// nested values. It does not alter VLAN or its shared update request shape.
// This is response representation, NOT writable-field authorization. Duplicate
// keys use encoding/json's last-value behavior; object ordering is not retained.
// Null/nonobject responses are refused. Identifier validation is conservative
// SDK route safety (ASCII letters/digits/hyphen/underscore), not UUID grammar.
// No response link traversal, polling or followup requests occur.
// Docs: https://docs.cloudsigma.com/en/latest/networking.html#get-single-vlan
func (s *VLANsService) GetRaw(ctx context.Context, id string) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	q, err := s.client.NewRequest(http.MethodGet, "vlans/"+id+"/", nil)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	r, err := s.client.Do(ctx, q, &out)
	if err != nil {
		return nil, r, err
	}
	if out == nil {
		return nil, r, fmt.Errorf("expected VLAN response object")
	}
	return out, r, nil
}
