package cloudsigma

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// IPWithSubscription is an opt-in IP response including its subscription
// reference, not the full Subscription entity (whose ID is a string).
// The embedded IP leaves the legacy IP layout and request encoding unchanged.
// VLANSubscription has the same documented numeric id/resource_uri reference
// shape. Absent and null subscription both decode to nil; an empty object
// decodes to a non-nil reference with zero fields. Missing/null nested scalar
// fields become zero values. Unknown fields are ignored. Use GetRaw when those
// distinctions or unmodeled fields must be preserved.
// This response type does not add subscription support to list readers.
type IPWithSubscription struct {
	IP
	Subscription *VLANSubscription `json:"subscription,omitempty"`
}

// GetWithSubscription retrieves an IP with its typed subscription reference.
// It accepts canonical dotted IPv4, matching GetRaw's bounded route policy;
// IPv6 route semantics remain unresolved. It does not follow response links.
// A null response is rejected with Response retained; decoding failures follow
// Client.Do's nil-Response convention. Only the first JSON value is decoded.
// Docs: https://docs.cloudsigma.com/en/latest/networking.html#get-single-ip
func (s *IPsService) GetWithSubscription(ctx context.Context, id string) (*IPWithSubscription, *Response, error) {
	if id == "" {
		return nil, nil, ErrEmptyArgument
	}
	parsed := net.ParseIP(id)
	if parsed == nil || parsed.To4() == nil || strings.Contains(id, ":") || parsed.String() != id {
		return nil, nil, fmt.Errorf("typed IP read requires canonical dotted IPv4")
	}
	req, err := s.client.NewRequest(http.MethodGet, "ips/"+id+"/", nil)
	if err != nil {
		return nil, nil, err
	}
	var ip *IPWithSubscription
	resp, err := s.client.Do(ctx, req, &ip)
	if err != nil {
		return nil, resp, err
	}
	if ip == nil {
		return nil, resp, fmt.Errorf("expected IP response object")
	}
	return ip, resp, nil
}
