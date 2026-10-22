package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Stop requests that a backfilling server stop, returning its raw acknowledgment.
// Stop is hard power-off, not graceful ACPI shutdown.
// HTTP 202 means accepted, not stopped or lifecycle completion. The documented
// default empty object is sent; no avoid/options, retry, polling or follow-up is
// performed. Unsafe identifiers are refused as SDK policy, not UUID grammar.
// Raw fields are preserved without schema validation or redaction. Shared
// Client.Do success, decoding and error rules apply.
// Docs: https://docs.cloudsigma.com/en/latest/backfilling.html#stop
// Request/response: https://docs.cloudsigma.com/en/latest/servers.html#stop
func (s *BServersService) Stop(ctx context.Context, id string) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodPost, "bservers/"+id+"/action/?do=stop", struct{}{})
	if err != nil {
		return nil, nil, err
	}
	var value map[string]json.RawMessage
	resp, err := s.client.Do(ctx, req, &value)
	if err != nil {
		return nil, resp, err
	}
	if value == nil {
		return nil, resp, fmt.Errorf("expected backfilling stop acknowledgment response object")
	}
	return value, resp, nil
}
