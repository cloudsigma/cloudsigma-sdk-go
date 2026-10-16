package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ACLDefinition is an opt-in presence-preserving ACL object. In particular,
// it can carry the documented grantees list missing from the legacy ACL model.
// Raw values preserve null, empty collections and numeric tokens. Representation
// does not establish field writability. Callers must not mutate maps or bytes
// concurrently with CreateDefinition or UpdateDefinition. Duplicate JSON keys follow encoding/json.
type ACLDefinition map[string]json.RawMessage

// CreateDefinition posts the documented objects envelope to /acls/. It preserves
// the complete response envelope, including nested grantees, rather than decoding
// into the legacy ACL model. Response.Meta is not populated; no links are followed.
// Nil batches/objects and known readonly top-level fields (owner, resource_uri,
// uuid) are rejected before dispatch. Empty nonnil objects/batches are serialized,
// not guaranteed valid server requests. Unknown fields are forwarded, not endorsed.
// Metadata follows the common replacement contract, not a merge operation.
// Null/nonobject responses are refused. Each returned raw value is caller-owned.
// Docs: https://docs.cloudsigma.com/en/latest/acls.html#creating
func (s *ACLsService) CreateDefinition(ctx context.Context, objects []ACLDefinition) (map[string]json.RawMessage, *Response, error) {
	if objects == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}
	snapshot := make([]ACLDefinition, len(objects))
	for i, object := range objects {
		if object == nil {
			return nil, nil, ErrEmptyPayloadNotAllowed
		}
		snapshot[i] = make(ACLDefinition, len(object))
		for k, v := range object {
			switch k {
			case "owner", "resource_uri", "uuid":
				return nil, nil, fmt.Errorf("readonly ACL definition field: %s", k)
			}
			if !json.Valid(v) {
				return nil, nil, fmt.Errorf("invalid ACL definition JSON")
			}
			snapshot[i][k] = append(json.RawMessage(nil), v...)
		}
	}
	req, err := s.client.NewRequest(http.MethodPost, "acls/", struct {
		Objects []ACLDefinition `json:"objects"`
	}{snapshot})
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	resp, err := s.client.Do(ctx, req, &out)
	if err != nil {
		return nil, resp, err
	}
	if out == nil {
		return nil, resp, fmt.Errorf("expected ACL create response object")
	}
	return out, resp, nil
}
