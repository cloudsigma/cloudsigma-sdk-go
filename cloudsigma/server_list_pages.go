package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// ServerListOptions contains only filters declared in the public server schema.
// Nil options omit query parameters. A nonnil zero value sends limit=0, meaning
// all resources under the public pagination contract; offset zero is omitted.
// Fields projection is intentionally absent: the inspected server/general docs
// do not establish a fields query contract. Negative pagination is SDK-rejected.
type ServerListOptions struct {
	ListOptions
	Name         string `url:"name,omitempty"`
	NameContains string `url:"name__contains,omitempty"`
	Tag          string `url:"tag,omitempty"`
	UUID         string `url:"uuid,omitempty"`
}

// ListSummary retrieves one page from /servers/ using the existing typed Server
// projection. It does not infer fields omitted by the summary response.
func (s *ServersService) ListSummary(ctx context.Context, opts *ServerListOptions) ([]Server, *Response, error) {
	return s.listPage(ctx, "servers/", opts)
}

// ListDetail retrieves one page from /servers/detail/. Existing List remains
// unchanged. No automatic traversal of next/previous URLs or retries occurs.
func (s *ServersService) ListDetail(ctx context.Context, opts *ServerListOptions) ([]Server, *Response, error) {
	return s.listPage(ctx, "servers/detail/", opts)
}
func (s *ServersService) listPage(ctx context.Context, path string, opts *ServerListOptions) ([]Server, *Response, error) {
	if opts != nil && (opts.Limit < 0 || opts.Offset < 0) {
		return nil, nil, fmt.Errorf("negative server pagination")
	}
	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(serversRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Meta != nil {
		resp.Meta = root.Meta
	}
	return root.values(), resp, nil
}
