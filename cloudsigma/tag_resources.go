package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListResourcesRaw reads one GET /tags/{id}/{resource_type}/ collection envelope.
// resourceType must be one of the documented servers, drives, ips or vlans.
// The complete JSON object, including meta and unknown fields, is preserved;
// Response.Meta is not populated. No pagination or response-link traversal is
// performed. Resource fields are opaque, not projected into shared write models.
// Each call owns its raw values. Null/nonobject responses are refused; duplicate
// keys follow encoding/json's last-value behavior. Identifier validation is
// conservative SDK route safety, not a public UUID grammar.
// Docs: https://docs.cloudsigma.com/en/latest/tags.html
func (s *TagsService) ListResourcesRaw(ctx context.Context, id, resourceType string) (map[string]json.RawMessage, *Response, error) {
	return s.ListResourcesPageRaw(ctx, id, resourceType, nil)
}

// ListResourcesPageRaw reads one tagged-resource collection page with its full raw
// envelope. Nil options omit pagination; a nonnil zero value sends limit=0.
// Negative pagination is rejected locally. Validation, response ownership and
// error behavior match ListResourcesRaw. No links are followed, Response.Meta is
// not populated, and resource fields are not schema-validated or redacted.
// Docs: https://docs.cloudsigma.com/en/latest/tags.html
func (s *TagsService) ListResourcesPageRaw(ctx context.Context, id, resourceType string, opts *ListOptions) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	switch resourceType {
	case "servers", "drives", "ips", "vlans":
	default:
		return nil, nil, fmt.Errorf("tag resource type must be servers, drives, ips or vlans")
	}
	if opts != nil && (opts.Limit < 0 || opts.Offset < 0) {
		return nil, nil, fmt.Errorf("negative tagged resources pagination")
	}
	path, err := addOptions("tags/"+id+"/"+resourceType+"/", opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	resp, err := s.client.Do(ctx, req, &out)
	if err != nil {
		return nil, resp, err
	}
	if out == nil {
		return nil, resp, fmt.Errorf("expected tagged resources response object")
	}
	return out, resp, nil
}
