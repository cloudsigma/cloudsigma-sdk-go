package cloudsigma

import (
	"context"
	"net/http"
)

// BServerMinimalCreateRequest contains the four fields in the documented minimal
// server creation request. All fields are always sent, including zero and empty
// values. The server validates values, capacity and account permissions.
type BServerMinimalCreateRequest struct {
	CPU         int64  `json:"cpu"`
	Mem         int64  `json:"mem"`
	Name        string `json:"name"`
	VNCPassword string `json:"vnc_password"`
}

// CreateMinimalRaw posts minimal backfilling server definitions and retains the
// complete raw response envelope. Nil input is encoded as an empty objects array;
// this does not imply the server accepts empty batches. HTTP 201 acknowledges
// creation, not running or readiness. No retries or follow-up requests are made.
// Docs: https://docs.cloudsigma.com/en/latest/backfilling.html#creating
// Minimal fields: https://docs.cloudsigma.com/en/latest/servers.html#creating
func (s *BServersService) CreateMinimalRaw(ctx context.Context, servers []BServerMinimalCreateRequest) (*BServers, *Response, error) {
	if servers == nil {
		servers = []BServerMinimalCreateRequest{}
	}
	body := struct {
		Objects []BServerMinimalCreateRequest `json:"objects"`
	}{Objects: servers}
	req, err := s.client.NewRequest(http.MethodPost, "bservers/", body)
	if err != nil {
		return nil, nil, err
	}
	value := new(BServers)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
