package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// GetPage reads one /ledger/ collection page with its complete raw envelope.
// Nil options omit pagination; a nonnil zero value requests limit=0 (all
// resources). Negative pagination is rejected by the SDK. Decimal strings and
// numeric lexemes are preserved, not interpreted. No automatic traversal or
// monetary computation is performed. Get remains unchanged.
// Docs: https://docs.cloudsigma.com/en/latest/pagination.html
func (s *LedgerService) GetPage(ctx context.Context, opts *ListOptions) (*Ledger, *Response, error) {
	if opts != nil && (opts.Limit < 0 || opts.Offset < 0) {
		return nil, nil, fmt.Errorf("negative ledger pagination")
	}
	path, err := addOptions("ledger/", opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(Ledger)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
