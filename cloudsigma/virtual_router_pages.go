package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// VirtualRoutersService provides opt-in raw public router reads.
type VirtualRoutersService service

// ListSummaryRaw retrieves one summary page as owned raw JSON object maps.
// Nil options omit query; nonnil options encode supplied Limit, including zero.
// Negative pagination is refused as SDK policy. Only numeric pagination metadata
// is promoted into Response.Meta; unknown envelope/meta fields are discarded.
// Object values preserve raw JSON, not writability or whole-document fidelity.
// Duplicate keys last win; key ordering is lost; one-value decoding does not
// validate trailing documents. No links are traversed or filters inferred.
func (s *VirtualRoutersService) ListSummaryRaw(ctx context.Context, opts *ListOptions) ([]map[string]json.RawMessage, *Response, error) {
	return s.listRawPage(ctx, "virtualrouters/", opts)
}

// ListDetailRaw retrieves one detailed virtual-router page. Existing service methods remain unchanged.
// Object fields are raw; envelope handling matches ListSummaryRaw. Missing/null
// objects decode to nil, empty arrays remain nonnil, null array elements remain nil.
func (s *VirtualRoutersService) ListDetailRaw(ctx context.Context, opts *ListOptions) ([]map[string]json.RawMessage, *Response, error) {
	return s.listRawPage(ctx, "virtualrouters/detail/", opts)
}
func (s *VirtualRoutersService) listRawPage(ctx context.Context, path string, opts *ListOptions) ([]map[string]json.RawMessage, *Response, error) {
	if opts != nil && (opts.Limit < 0 || opts.Offset < 0) {
		return nil, nil, fmt.Errorf("negative virtual-router pagination")
	}
	path, e := addOptions(path, opts)
	if e != nil {
		return nil, nil, e
	}
	q, e := s.client.NewRequest(http.MethodGet, path, nil)
	if e != nil {
		return nil, nil, e
	}
	root := new(struct {
		Meta    *Meta                        `json:"meta"`
		Objects []map[string]json.RawMessage `json:"objects"`
	})
	r, e := s.client.Do(ctx, q, root)
	if e != nil {
		return nil, r, e
	}
	if root.Meta != nil {
		r.Meta = root.Meta
	}
	return root.Objects, r, nil
}
