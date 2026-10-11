package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// ListSummary retrieves one IP summary page. Nil options omit query parameters;
// nonnil zero options send limit=0 (all resources). Negative values are refused
// as SDK policy. No remote pagination link is followed. Filters/projection are
// deliberately not added: this bounded slice adds only pagination.
func (s *IPsService) ListSummary(ctx context.Context, opts *ListOptions) ([]IP, *Response, error) {
	return s.listPage(ctx, "ips/", opts)
}

// ListDetail retrieves one detailed IP page. Legacy List/Get remain unchanged.
// Only modeled fields are retained; unknown fields and pagination links are not.
func (s *IPsService) ListDetail(ctx context.Context, opts *ListOptions) ([]IP, *Response, error) {
	return s.listPage(ctx, "ips/detail/", opts)
}
func (s *IPsService) listPage(ctx context.Context, path string, opts *ListOptions) ([]IP, *Response, error) {
	if opts != nil && (opts.Limit < 0 || opts.Offset < 0) {
		return nil, nil, fmt.Errorf("negative IP pagination")
	}
	path, e := addOptions(path, opts)
	if e != nil {
		return nil, nil, e
	}
	q, e := s.client.NewRequest(http.MethodGet, path, nil)
	if e != nil {
		return nil, nil, e
	}
	root := new(ipsRoot)
	r, e := s.client.Do(ctx, q, root)
	if e != nil {
		return nil, r, e
	}
	if root.Meta != nil {
		r.Meta = root.Meta
	}
	return root.IPs, r, nil
}
