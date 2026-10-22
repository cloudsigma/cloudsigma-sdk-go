package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Start requests that a backfilling server start, returning its raw acknowledgment.
// HTTP 202 means accepted, not running or lifecycle completion. The documented
// default empty object is sent; no avoid/options, retry, polling or follow-up is
// performed. Unsafe identifiers are refused as SDK policy, not UUID grammar.
// Raw fields are preserved without schema validation or redaction. Shared
// Client.Do success, decoding and error rules apply.
// Redirects are refused with ErrBServerStartRedirect; the initial POST outcome
// is unknown and must not be blindly retried.
// Docs: https://docs.cloudsigma.com/en/latest/backfilling.html#start
// Request/response: https://docs.cloudsigma.com/en/latest/servers.html#start
func (s *BServersService) Start(ctx context.Context, id string) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodPost, "bservers/"+id+"/action/?do=start", struct{}{})
	if err != nil {
		return nil, nil, err
	}
	var value map[string]json.RawMessage
	resp, err := s.client.doResourceAction(ctx, req, &value, ErrBServerStartRedirect)
	if err != nil {
		return nil, resp, err
	}
	if value == nil {
		return nil, resp, fmt.Errorf("expected backfilling start acknowledgment response object")
	}
	return value, resp, nil
}
