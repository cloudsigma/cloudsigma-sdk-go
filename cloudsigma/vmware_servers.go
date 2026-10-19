package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// VMwareServersService provides the documented VMware list read only.
type VMwareServersService service

// VMwareServers preserves a dynamic objects/meta envelope without schema validation
// or redaction. Each List owns fresh raw values; ordinary map copies may alias.
type VMwareServers map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *VMwareServers) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("vmware_servers response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// List reads one GET /vmware_servers/ response without query or traversal.
// No singleton, lifecycle or runtime feature-availability contract is inferred.
// Docs: https://docs.cloudsigma.com/en/latest/servers_vmware.html
func (s *VMwareServersService) List(ctx context.Context) (*VMwareServers, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "vmware_servers/", nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(VMwareServers)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
