package cloudsigma

import (
	"context"
	"net/http"
)

// DeleteMany requests deletion of the specified schedulers in one collection
// request. HTTP 204 acknowledges deletion started, not completed. Empty input
// is rejected as SDK policy, not a documented server requirement. Identifiers
// use the same conservative local policy as Delete. Order and duplicates are
// preserved; duplicate-ID and partial-failure semantics are unspecified.
// No separate unlink/backup-delete requests, polling or retries are issued.
// Server-side cascade and ownership effects are unspecified. Callers must not
// mutate ids concurrently. Docs: https://docs.cloudsigma.com/en/latest/backup_schedulers.html
func (s *BackupSchedulersService) DeleteMany(ctx context.Context, ids []string) (*Response, error) {
	if len(ids) == 0 {
		return nil, ErrEmptyArgument
	}
	type reference struct {
		UUID string `json:"uuid"`
	}
	body := struct {
		Objects []reference `json:"objects"`
	}{Objects: make([]reference, len(ids))}
	for i, id := range ids {
		if err := schedulerWriteID(id); err != nil {
			return nil, err
		}
		body.Objects[i] = reference{UUID: id}
	}
	req, err := s.client.NewRequest(http.MethodDelete, "backupschedulers/", body)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}
