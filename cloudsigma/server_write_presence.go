package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ServerDefinition is a caller-owned raw JSON server object. It preserves
// explicit false/zero/empty/null and unknown keys. The public docs establish
// the objects envelope for create and singleton body for update; they do not
// establish writability of every response field. Known top-level readonly keys
// are rejected, even when null. Unknown keys are not guaranteed writable.
// Callers must not mutate maps or raw bytes concurrently with these methods.
type ServerDefinition map[string]json.RawMessage

func validateServerDefinitionID(id string) error {
	if id == "" {
		return ErrEmptyArgument
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return fmt.Errorf("invalid server identifier")
		}
	}
	return nil
}
func copyServerDefinition(v ServerDefinition) (ServerDefinition, error) {
	if v == nil {
		return nil, ErrEmptyPayloadNotAllowed
	}
	o := make(ServerDefinition, len(v))
	for k, b := range v {
		switch k {
		case "jobs", "owner", "resource_uri", "runtime", "status", "uuid":
			return nil, fmt.Errorf("readonly server definition field: %s", k)
		}
		if !json.Valid(b) {
			return nil, fmt.Errorf("invalid server definition JSON")
		}
		o[k] = append(json.RawMessage(nil), b...)
	}
	return o, nil
}

// CreateDefinition sends the documented objects envelope with presence-preserving
// raw server objects and decodes the collection objects response into []Server.
// Nil batches/objects are rejected; nonnil empty batches/objects are serialized.
// This is SDK representation policy, not a claim the server accepts empty writes.
func (s *ServersService) CreateDefinition(ctx context.Context, objects []ServerDefinition) ([]Server, *Response, error) {
	if objects == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}
	snap := make([]ServerDefinition, len(objects))
	for i, v := range objects {
		x, e := copyServerDefinition(v)
		if e != nil {
			return nil, nil, e
		}
		snap[i] = x
	}
	req, e := s.client.NewRequest(http.MethodPost, "servers/", struct {
		Objects []ServerDefinition `json:"objects"`
	}{snap})
	if e != nil {
		return nil, nil, e
	}
	root := new(serversRoot)
	resp, e := s.client.Do(ctx, req, root)
	if e != nil {
		return nil, resp, e
	}
	return root.values(), resp, nil
}

// UpdateDefinition sends the documented singleton PUT body without mutating the
// caller. It does not infer server-side field writability or running-state rules.
func (s *ServersService) UpdateDefinition(ctx context.Context, id string, definition ServerDefinition) (*Server, *Response, error) {
	if e := validateServerDefinitionID(id); e != nil {
		return nil, nil, e
	}
	snap, e := copyServerDefinition(definition)
	if e != nil {
		return nil, nil, e
	}
	req, e := s.client.NewRequest(http.MethodPut, fmt.Sprintf("servers/%s/", id), snap)
	if e != nil {
		return nil, nil, e
	}
	out := new(serverResponse)
	resp, e := s.client.Do(ctx, req, out)
	if e != nil {
		return nil, resp, e
	}
	return (*Server)(out), resp, nil
}
