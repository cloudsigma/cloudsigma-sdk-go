package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ServerCloneRequest is an optional JSON object. Documented keys are name and
// random_vnc_password. Raw fields preserve explicit empty/false/null and unknown
// values; this is serialization fidelity, not a promise unknown keys are valid.
type ServerCloneRequest map[string]json.RawMessage

// Clone requests the documented server clone and decodes its singleton server
// definition using the existing private server decoder. Nil options omit the
// optional body; an empty map sends {}. No required fields are invented.
// HTTP 202 is acceptance, not completion. No polling or live followups occur.
// The existing Server model ignores other unmodeled fields; this is
// not a raw full-response interface. Caller options must not mutate concurrently.
// IDs use ASCII letters/digits/hyphen/underscore SDK policy, not UUID grammar.
// Docs: https://docs.cloudsigma.com/en/latest/servers.html#cloning
func (s *ServersService) Clone(ctx context.Context, id string, options ServerCloneRequest) (*Server, *Response, error) {
	return s.cloneAvoid(ctx, id, options, nil)
}

func (s *ServersService) cloneAvoid(ctx context.Context, id string, options ServerCloneRequest, avoid []string) (*Server, *Response, error) {
	if id == "" {
		return nil, nil, ErrEmptyArgument
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return nil, nil, fmt.Errorf("invalid server identifier")
		}
	}
	suffix, err := serverAvoidQuery(avoid)
	if err != nil {
		return nil, nil, err
	}
	var body interface{}
	if options != nil {
		snapshot := make(ServerCloneRequest, len(options))
		for k, v := range options {
			if !json.Valid(v) {
				return nil, nil, fmt.Errorf("invalid clone option JSON")
			}
			snapshot[k] = append(json.RawMessage(nil), v...)
		}
		body = snapshot
	}
	req, err := s.client.NewRequest(http.MethodPost, fmt.Sprintf("servers/%s/action/?do=clone", id)+suffix, body)
	if err != nil {
		return nil, nil, err
	}
	result := new(serverResponse)
	resp, err := s.client.Do(ctx, req, result)
	if err != nil {
		return nil, resp, err
	}
	return (*Server)(result), resp, nil
}
