package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// CloseConsole sends the public close_console action and returns its documented
// action/result/uuid acknowledgment (no console URL). HTTP 202 means accepted,
// not confirmed closure. The public heading says serial console while prose
// says VNC tunnel; this method does not infer equivalence to close_vnc.
// No connections, passwords, polling, retries or followups are performed.
// IDs use nonempty ASCII letters/digits/hyphen/underscore as SDK route policy,
// not public UUID grammar. Existing methods are unaffected.
// Redirects are refused with ErrServerConsoleCloseRedirect; the initial POST
// outcome is unknown. Callers must not blindly retry.
// Docs: https://docs.cloudsigma.com/en/latest/servers.html#close-serial-console-connection
func (s *ServersService) CloseConsole(ctx context.Context, id string) (*ServerAction, *Response, error) {
	if id == "" {
		return nil, nil, ErrEmptyArgument
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return nil, nil, fmt.Errorf("invalid server identifier")
		}
	}
	req, err := s.client.NewRequest(http.MethodPost, fmt.Sprintf("servers/%s/action/?do=close_console", id), struct{}{})
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
