package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// UpdateDefinition sends a presence-preserving singleton object to PUT /acls/{id}/
// and retains the full response, including grantees. Unlike CreateDefinition,
// it forwards even owner, resource_uri and uuid to represent the complete public
// update example. This does not assert that readonly fields are writable or
// establish whether the server ignores, rejects or checks them. Unknown fields
// are likewise forwarded, not endorsed. The readonly-input contract remains open.
//
// Nil definitions and invalid raw JSON are refused before dispatch. The identifier
// uses the SDK conservative route-segment policy, not a public UUID grammar.
// Empty objects, null fields and wrong field types are represented, not validated
// against the server schema. Callers own full metadata replacement: there is no
// read/merge, atomicity guarantee or permission-policy inference. Do not mutate
// definition maps or bytes concurrently with this call. Input bytes are copied;
// returned raw values are caller-owned. Null/nonobject responses are refused.
// Response.Meta is not populated and no links are followed.
// Docs: https://docs.cloudsigma.com/en/latest/acls.html#editing
func (s *ACLsService) UpdateDefinition(ctx context.Context, id string, definition ACLDefinition) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	if definition == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}
	snapshot := make(ACLDefinition, len(definition))
	for k, v := range definition {
		if !json.Valid(v) {
			return nil, nil, fmt.Errorf("invalid ACL update definition JSON")
		}
		snapshot[k] = append(json.RawMessage(nil), v...)
	}
	req, err := s.client.NewRequest(http.MethodPut, "acls/"+id+"/", snapshot)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	resp, err := s.client.Do(ctx, req, &out)
	if err != nil {
		return nil, resp, err
	}
	if out == nil {
		return nil, resp, fmt.Errorf("expected ACL update response object")
	}
	return out, resp, nil
}
