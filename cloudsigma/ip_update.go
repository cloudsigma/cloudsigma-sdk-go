package cloudsigma

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"unicode/utf8"
)

// IPUpdateRequest submits FULL metadata replacement, not a merge. Include every
// existing pair you wish to retain; omitted pairs are deleted per public docs.
// A nonnil empty map requests replacement with no pairs (not live-verified).
// Nil request/map are refused as SDK policy. Public docs say values "should be
// strings"; this API deliberately enforces strings via its type, without coercion.
// Keys are limited to 32 Unicode code points, not bytes or grapheme clusters,
// an SDK interpretation of the public "characters" limit. Invalid UTF-8 in keys
// or values is refused rather than silently repaired; empty keys are permitted.
// Callers must not mutate the map concurrently. No automatic GET/merge occurs.
// Contract: https://docs.cloudsigma.com/en/latest/meta.html (retained lines15/17).
type IPUpdateRequest struct {
	Meta map[string]string `json:"meta"`
}

// Update sends the documented singleton metadata PUT and decodes a typed IP.
// This bounded method accepts canonical dotted IPv4 only, matching public
// examples. IPv6, zones, CIDRs and UUIDs are not established by those examples
// and are refused as SDK policy, not a claim about server support. Existing Get
// and List are unchanged. Unknown response fields remain omitted.
// Docs: https://docs.cloudsigma.com/en/latest/networking.html
func (s *IPsService) Update(ctx context.Context, address string, body *IPUpdateRequest) (*IP, *Response, error) {
	if address == "" {
		return nil, nil, ErrEmptyArgument
	}
	parsed := net.ParseIP(address)
	if parsed == nil || parsed.To4() == nil || strings.Contains(address, ":") || parsed.String() != address {
		return nil, nil, fmt.Errorf("IP update requires canonical dotted IPv4")
	}
	if body == nil || body.Meta == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}
	snapshot := IPUpdateRequest{Meta: make(map[string]string, len(body.Meta))}
	for k, v := range body.Meta {
		if !utf8.ValidString(k) || utf8.RuneCountInString(k) > 32 || !utf8.ValidString(v) {
			return nil, nil, fmt.Errorf("invalid IP metadata key or UTF-8")
		}
		snapshot.Meta[k] = v
	}
	q, e := s.client.NewRequest(http.MethodPut, "ips/"+address+"/", snapshot)
	if e != nil {
		return nil, nil, e
	}
	out := new(IP)
	r, e := s.client.Do(ctx, q, out)
	if e != nil {
		return nil, r, e
	}
	return out, r, nil
}
