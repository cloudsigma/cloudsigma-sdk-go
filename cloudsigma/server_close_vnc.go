package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// CloseVNC sends the documented KVM close_vnc action with {}.
// HTTP 202 acknowledges acceptance, not readiness or completed closure.
// No connection, password, polling, retries or followups are performed.
// IDs use conservative ASCII letters/digits/hyphen/underscore SDK policy,
// not public UUID grammar. No VMware action applicability is inferred.
// Redirects are refused with ErrServerConsoleCloseRedirect; the initial POST
// outcome is unknown. Callers must not blindly retry.
// Docs: https://docs.cloudsigma.com/en/latest/servers_kvm.html
func (s *ServersService) CloseVNC(ctx context.Context, id string) (*ServerAction, *Response, error) {
	if id == "" {
		return nil, nil, ErrEmptyArgument
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return nil, nil, fmt.Errorf("invalid server identifier")
		}
	}
	req, err := s.client.NewRequest(http.MethodPost, fmt.Sprintf("servers/%s/action/?do=close_vnc", id), struct{}{})
	if err != nil {
		return nil, nil, err
	}
	result := new(ServerAction)
	resp, err := s.client.doResourceAction(ctx, req, result, ErrServerConsoleCloseRedirect)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}
