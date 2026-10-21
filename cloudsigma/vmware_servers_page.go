package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// ListPage reads one /vmware_servers/ collection page with its complete raw envelope.
// Nil options omit pagination; a nonnil zero value requests limit=0 (all
// resources). Negative pagination is rejected by the SDK. Nested values and
// numeric lexemes are preserved, not interpreted. No automatic traversal or
// lifecycle operation is performed. List remains unchanged.
// Docs: https://docs.cloudsigma.com/en/latest/pagination.html
func (s *VMwareServersService) ListPage(ctx context.Context, opts *ListOptions) (*VMwareServers, *Response, error) {
	if opts != nil && (opts.Limit < 0 || opts.Offset < 0) {
		return nil, nil, fmt.Errorf("negative vmware_servers pagination")
	}
	path, err := addOptions("vmware_servers/", opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(VMwareServers)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
