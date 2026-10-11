package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func schedulerWriteID(id string) error {
	if id == "" {
		return ErrEmptyArgument
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return fmt.Errorf("invalid backup scheduler identifier")
		}
	}
	return nil
}

// Update edits a scheduler using a caller-supplied JSON object. Raw fields
// retain explicit false/zero/empty/null and unknown keys. This validates JSON,
// not server writability or requiredness; callers must not mutate input
// concurrently. Identifiers follow the conservative single-segment SDK policy.
// Docs: https://docs.cloudsigma.com/en/latest/backup_schedulers.html
func (s *BackupSchedulersService) Update(ctx context.Context, id string, definition BackupScheduler) (BackupScheduler, *Response, error) {
	if err := schedulerWriteID(id); err != nil {
		return nil, nil, err
	}
	if definition == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}
	snapshot := make(BackupScheduler, len(definition))
	for k, v := range definition {
		if !json.Valid(v) {
			return nil, nil, fmt.Errorf("invalid scheduler definition JSON")
		}
		snapshot[k] = append(json.RawMessage(nil), v...)
	}
	req, err := s.client.NewRequest(http.MethodPut, fmt.Sprintf("backupschedulers/%s/", id), snapshot)
	if err != nil {
		return nil, nil, err
	}
	var result BackupScheduler
	resp, err := s.client.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}

// Delete requests deletion of one scheduler. Documented HTTP204 means deletion
// started, NOT confirmed absence. It does not issue separate unlink or backup-delete requests; server-side
// cascade effects are unspecified. It does not
// poll, retry or translate404 to success; existing HTTP errors remain errors.
func (s *BackupSchedulersService) Delete(ctx context.Context, id string) (*Response, error) {
	if err := schedulerWriteID(id); err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(http.MethodDelete, fmt.Sprintf("backupschedulers/%s/", id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}
