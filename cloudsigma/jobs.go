package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// JobsService reads the progress records referenced by drive and server clones.
// Docs: https://docs.cloudsigma.com/en/latest/jobs.html
type JobsService service

// Get retrieves one job, including nested children and data, without polling or
// interpreting completion. Raw values preserve field presence and numeric tokens
// and are caller-owned; field types are not schema-validated. Null and nonobject
// responses are refused. The identifier follows the SDK conservative segment
// policy, not a claim about public UUID grammar. No resource links are followed.
func (s *JobsService) Get(ctx context.Context, id string) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, "jobs/"+id+"/", nil)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	resp, err := s.client.Do(ctx, req, &out)
	if err != nil {
		return nil, resp, err
	}
	if out == nil {
		return nil, resp, fmt.Errorf("expected job response object")
	}
	return out, resp, nil
}
