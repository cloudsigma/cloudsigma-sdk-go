package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// DriveSchedulerBinding describes the documented scheduler link response.
// Drive retains the existing private license decoder; BackupScheduler is the
// newly observed link, not proof that an asynchronous backup has completed.
type DriveSchedulerBinding struct {
	*Drive
	BackupScheduler *ResourceLink `json:"backup_scheduler"`
}

// SetScheduler links a backup policy to a drive. It does not implement unlink,
// scheduler creation, or polling. HTTP 202 means accepted, not complete.
// Both identifiers must be nonempty ASCII letters/digits/hyphen/underscore:
// conservative SDK input safety, not a claim about every server-valid ID.
// Redirects are refused with ErrDriveSetSchedulerRedirect. The initial POST
// outcome is unknown; callers must not blindly retry.
// Public docs: https://docs.cloudsigma.com/en/latest/backup_schedulers.html
func (s *DrivesService) SetScheduler(ctx context.Context, driveID, policyID string) (*DriveSchedulerBinding, *Response, error) {
	for _, id := range []string{driveID, policyID} {
		if id == "" {
			return nil, nil, ErrEmptyArgument
		}
		for _, c := range id {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
				return nil, nil, fmt.Errorf("invalid scheduler binding identifier")
			}
		}
	}
	payload := struct {
		Policy string `json:"backup_policy_uuid"`
	}{policyID}
	req, err := s.client.NewRequest(http.MethodPost, fmt.Sprintf("drives/%s/action/?do=set_scheduler", driveID), payload)
	if err != nil {
		return nil, nil, err
	}
	var raw json.RawMessage
	resp, err := s.client.doResourceAction(ctx, req, &raw, ErrDriveSetSchedulerRedirect)
	if err != nil {
		return nil, resp, err
	}
	var drive driveResponse
	if err = json.Unmarshal(raw, &drive); err != nil {
		return nil, nil, err
	}
	var link struct {
		BackupScheduler *ResourceLink `json:"backup_scheduler"`
	}
	if err = json.Unmarshal(raw, &link); err != nil {
		return nil, nil, err
	}
	return &DriveSchedulerBinding{Drive: (*Drive)(&drive), BackupScheduler: link.BackupScheduler}, resp, nil
}
