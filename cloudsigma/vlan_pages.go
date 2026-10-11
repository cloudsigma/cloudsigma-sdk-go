package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// ListSummary retrieves one VLAN summary page. Nil options omit query parameters;
// nonnil zero options send limit=0 (all resources). Negative values are refused
// as SDK policy. No remote pagination link is followed. Filters/projection are
// deliberately not added: networking's final schema is not assumed VLAN-specific.
func (s *VLANsService) ListSummary(ctx context.Context, opts *ListOptions) ([]VLAN, *Response, error) {
	return s.listPage(ctx, "vlans/", opts)
}

// ListDetail retrieves one detailed VLAN page. Legacy List/Get remain unchanged.
// Only modeled fields are retained; unknown fields and pagination links are not.
func (s *VLANsService) ListDetail(ctx context.Context, opts *ListOptions) ([]VLAN, *Response, error) {
	return s.listPage(ctx, "vlans/detail/", opts)
}
func (s *VLANsService) listPage(ctx context.Context, path string, opts *ListOptions) ([]VLAN, *Response, error) {
	if opts != nil && (opts.Limit < 0 || opts.Offset < 0) {
		return nil, nil, fmt.Errorf("negative VLAN pagination")
	}
	path, e := addOptions(path, opts)
	if e != nil {
		return nil, nil, e
	}
	q, e := s.client.NewRequest(http.MethodGet, path, nil)
	if e != nil {
		return nil, nil, e
	}
	root := new(vlansRoot)
	r, e := s.client.Do(ctx, q, root)
	if e != nil {
		return nil, r, e
	}
	if root.Meta != nil {
		r.Meta = root.Meta
	}
	return root.VLANs, r, nil
}
