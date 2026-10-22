package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// TunnelsService provides bounded virtual router VPN tunnel operations.
// Docs: https://docs.cloudsigma.com/en/latest/virtualrouters_vpn_tunnels.html
type TunnelsService service

// Delete requests deletion of a single VPN tunnel. HTTP 204 acknowledges that
// deletion started, not that it completed. No polling or retries are performed.
// Unsafe route identifiers are refused as SDK policy, not a public UUID grammar.
// Client transport, authentication and error handling remain unchanged.
func (s *TunnelsService) Delete(ctx context.Context, id string) (*Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(http.MethodDelete, fmt.Sprintf("tunnels/%s/", id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}
