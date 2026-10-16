package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetRaw retrieves a complete ACL singleton, including the documented grantees
// omitted by the legacy ACL model. It does not change Get or the shared model.
// Raw values retain field presence and numeric tokens and are caller-owned.
// Null/nonobject responses are refused; field types are not schema-validated.
// The identifier uses the SDK conservative segment policy, not public UUID grammar.
// Response.Meta is not populated and no links are followed. Reading readonly
// fields does not authorize forwarding them in a subsequent write.
// Docs: https://docs.cloudsigma.com/en/latest/acls.html#list-single-acl
func (s *ACLsService) GetRaw(ctx context.Context, id string) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, "acls/"+id+"/", nil)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	resp, err := s.client.Do(ctx, req, &out)
	if err != nil {
		return nil, resp, err
	}
	if out == nil {
		return nil, resp, fmt.Errorf("expected ACL response object")
	}
	return out, resp, nil
}
