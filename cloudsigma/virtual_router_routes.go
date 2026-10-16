package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// VirtualRouterRoutesService reads documented virtual-router routes.
type VirtualRouterRoutesService service

// GetRaw retrieves a single virtual-router route as an owned JSON object, preserving every
// field's raw JSON representation, including null, empty lists and unknown
// nested values. Existing models and write request shapes are unchanged.
// This is response representation, NOT writable-field authorization. Duplicate
// keys use encoding/json's last-value behavior; object ordering is not retained.
// Null/nonobject responses are refused. Identifier validation is conservative
// SDK route safety (ASCII letters/digits/hyphen/underscore), not UUID grammar.
// No response link traversal, polling or followup requests occur.
// Docs: https://docs.cloudsigma.com/en/latest/virtualrouters_routes.html#list-single-route
func (s *VirtualRouterRoutesService) GetRaw(ctx context.Context, id string) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	q, err := s.client.NewRequest(http.MethodGet, "routes/"+id+"/", nil)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	r, err := s.client.Do(ctx, q, &out)
	if err != nil {
		return nil, r, err
	}
	if out == nil {
		return nil, r, fmt.Errorf("expected virtual-router route response object")
	}
	return out, r, nil
}
