package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// ServerVNCAction is the documented KVM open_vnc acknowledgment.
// VNCURL is data only; the SDK never connects or sends passwords.
type ServerVNCAction struct {
	Action string `json:"action"`
	Result string `json:"result"`
	UUID   string `json:"uuid"`
	VNCURL string `json:"vnc_url"`
}

// OpenVNC sends the documented KVM open_vnc action with {}.
// HTTP 202 acknowledges acceptance, not readiness or completed closure.
// No connection, password, polling, retries or followups are performed.
// IDs use conservative ASCII letters/digits/hyphen/underscore SDK policy,
// not public UUID grammar. No VMware action applicability is inferred.
// Docs: https://docs.cloudsigma.com/en/latest/servers_kvm.html
func (s *ServersService) OpenVNC(ctx context.Context, id string) (*ServerVNCAction, *Response, error) {
	if id == "" {
		return nil, nil, ErrEmptyArgument
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return nil, nil, fmt.Errorf("invalid server identifier")
		}
	}
	req, err := s.client.NewRequest(http.MethodPost, fmt.Sprintf("servers/%s/action/?do=open_vnc", id), struct{}{})
	if err != nil {
		return nil, nil, err
	}
	result := new(ServerVNCAction)
	resp, err := s.client.Do(ctx, req, result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}
