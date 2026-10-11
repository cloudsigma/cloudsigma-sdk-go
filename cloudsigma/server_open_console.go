package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// ServerConsoleAction is the documented open_console acknowledgment. ConsoleURL
// is returned as data only; the SDK never connects to it or sends a password.
type ServerConsoleAction struct {
	Action     string `json:"action"`
	Result     string `json:"result"`
	UUID       string `json:"uuid"`
	ConsoleURL string `json:"console_url"`
}

// OpenConsole requests a serial-console connection using the public server
// action. HTTP 202 means accepted/execution proceeding, not completed or ready.
// No polling, connection, retries or lifecycle followups are performed.
// Identifiers use a conservative nonempty ASCII alnum/hyphen/underscore SDK
// policy, not a guarantee of server-valid UUID syntax.
// Docs: https://docs.cloudsigma.com/en/latest/servers.html#open-serial-console-connection
func (s *ServersService) OpenConsole(ctx context.Context, id string) (*ServerConsoleAction, *Response, error) {
	if id == "" {
		return nil, nil, ErrEmptyArgument
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return nil, nil, fmt.Errorf("invalid server identifier")
		}
	}
	req, err := s.client.NewRequest(http.MethodPost, fmt.Sprintf("servers/%s/action/?do=open_console", id), struct{}{})
	if err != nil {
		return nil, nil, err
	}
	result := new(ServerConsoleAction)
	resp, err := s.client.Do(ctx, req, result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}
