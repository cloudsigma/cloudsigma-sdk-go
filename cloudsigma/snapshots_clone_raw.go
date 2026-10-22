package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CloneRaw requests a default clone of a snapshot to a drive and returns the raw
// drive definition. HTTP 202 means accepted, not that the drive is ready.
// The documented empty object is sent; naming is left to the server profile.
// No overrides, avoid query, retry, polling or follow-up are performed. Unsafe
// identifiers are refused as SDK policy, not UUID grammar. Raw fields are
// preserved without schema validation or redaction. Shared Client.Do success,
// decoding and error rules apply.
// Redirects are refused with ErrSnapshotCloneRedirect; the initial POST outcome
// is unknown and must not be blindly retried.
// Docs: https://docs.cloudsigma.com/en/latest/snapshots.html#cloning
func (s *SnapshotsService) CloneRaw(ctx context.Context, id string) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodPost, "snapshots/"+id+"/action/?do=clone", struct{}{})
	if err != nil {
		return nil, nil, err
	}
	var value map[string]json.RawMessage
	resp, err := s.client.doResourceAction(ctx, req, &value, ErrSnapshotCloneRedirect)
	if err != nil {
		return nil, resp, err
	}
	if value == nil {
		return nil, resp, fmt.Errorf("expected snapshot clone drive response object")
	}
	return value, resp, nil
}
