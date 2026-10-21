package cloudsigma

import (
	"context"
	"fmt"
)

// ListSummaryPageRaw retrieves one /snapshots/ page with its complete raw envelope.
// Nil options omit pagination. A nonnil zero value sends limit=0 (all resources)
// and omits offset=0. Negative pagination is rejected by the SDK. No automatic
// traversal, retries, filtering, or lifecycle completion is implied.
// Existing ListSummaryRaw and ListDetailRaw remain unchanged.
// Docs: https://docs.cloudsigma.com/en/latest/pagination.html
func (s *SnapshotsService) ListSummaryPageRaw(ctx context.Context, opts *ListOptions) (*SnapshotRaw, *Response, error) {
	if opts != nil && (opts.Limit < 0 || opts.Offset < 0) {
		return nil, nil, fmt.Errorf("negative snapshot pagination")
	}
	path, err := addOptions("snapshots/", opts)
	if err != nil {
		return nil, nil, err
	}
	return s.listCollectionRaw(ctx, path)
}
