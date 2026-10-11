package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
	"unicode/utf8"
)

// VLANMetadataUpdateRequest submits FULL metadata replacement, not a merge. Include every
// existing pair you wish to retain; omitted pairs are deleted per public docs.
// A nonnil empty map requests replacement with no pairs (not live-verified).
// Nil request/map are refused as SDK policy. Public docs say values "should be
// strings"; this API deliberately enforces strings via its type, without coercion.
// Keys are limited to 32 Unicode code points, not bytes or grapheme clusters,
// an SDK interpretation of the public "characters" limit. Invalid UTF-8 in keys
// or values is refused rather than silently repaired; empty keys are permitted.
// Callers must not mutate the map concurrently. No automatic GET/merge occurs.
// Contract: https://docs.cloudsigma.com/en/latest/meta.html (retained lines15/17).
type VLANMetadataUpdateRequest struct {
	Meta map[string]string `json:"meta"`
}

// UpdateMetadata sends only metadata in a singleton PUT. It preserves an empty
// map as {"meta":{}}. Legacy Update/Get and their serialization remain unchanged.
// Identifiers allow ASCII letters/digits/hyphen/underscore as conservative SDK
// route safety, not public UUID grammar. No read-modify-write or atomicity claim.
// Unknown response fields remain omitted; no live clearing guarantee is made.
// Docs: https://docs.cloudsigma.com/en/latest/networking.html#editing
func (s *VLANsService) UpdateMetadata(ctx context.Context, address string, body *VLANMetadataUpdateRequest) (*VLAN, *Response, error) {
	if err := validateServerDefinitionID(address); err != nil {
		return nil, nil, err
	}
	if body == nil || body.Meta == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}
	snapshot := VLANMetadataUpdateRequest{Meta: make(map[string]string, len(body.Meta))}
	for k, v := range body.Meta {
		if !utf8.ValidString(k) || utf8.RuneCountInString(k) > 32 || !utf8.ValidString(v) {
			return nil, nil, fmt.Errorf("invalid VLAN metadata key or UTF-8")
		}
		snapshot.Meta[k] = v
	}
	q, e := s.client.NewRequest(http.MethodPut, "vlans/"+address+"/", snapshot)
	if e != nil {
		return nil, nil, e
	}
	out := new(VLAN)
	r, e := s.client.Do(ctx, q, out)
	if e != nil {
		return nil, r, e
	}
	return out, r, nil
}
