package cloudsigma

import (
	"context"
	"net/http"
)

// ListSummaryRaw reads one GET /snapshots/ response, retaining the complete
// objects/meta envelope. It does not traverse pages or infer filter options.
// SnapshotRaw validates only the outer object; raw data is not redacted.
// Docs: https://docs.cloudsigma.com/en/latest/snapshots.html
func (s *SnapshotsService) ListSummaryRaw(ctx context.Context) (*SnapshotRaw, *Response, error) {
	return s.listCollectionRaw(ctx, "snapshots/")
}

// ListDetailRaw reads one GET /snapshots/detail/ response, retaining the complete
// objects/meta envelope. Existing List and mutation serialization are unchanged.
// Each call owns fresh data; ordinary copies may alias. Existing Do semantics apply.
func (s *SnapshotsService) ListDetailRaw(ctx context.Context) (*SnapshotRaw, *Response, error) {
	return s.listCollectionRaw(ctx, "snapshots/detail/")
}

func (s *SnapshotsService) listCollectionRaw(ctx context.Context, path string) (*SnapshotRaw, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(SnapshotRaw)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
