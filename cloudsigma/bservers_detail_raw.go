package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// ListDetailRaw reads one detailed backfilling server collection. Nil options
// omit pagination; zero limit requests all resources. It preserves the raw
// envelope without schema validation or following pagination links.
// Docs: https://docs.cloudsigma.com/en/latest/backfilling.html#detailed-listing
func (s *BServersService) ListDetailRaw(ctx context.Context, opts *ListOptions) (*BServers, *Response, error) {
	if opts != nil && (opts.Limit < 0 || opts.Offset < 0) {
		return nil, nil, fmt.Errorf("negative bservers pagination")
	}
	path, err := addOptions("bservers/detail/", opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(BServers)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
