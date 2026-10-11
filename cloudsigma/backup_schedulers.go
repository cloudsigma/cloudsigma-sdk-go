package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// BackupScheduler retains the public scheduler object's fields without losing
// explicit zero/false/null or extensible nested schedule/retention policies.
// This read representation does not establish field writability.
type BackupScheduler map[string]json.RawMessage

// BackupSchedulerListOptions selects a documented type and offset page, with
// standard offset pagination. Empty Type means no type filter.
type BackupSchedulerListOptions struct {
	Type string `url:"type,omitempty"`
	// Fields projection is documented for List only. ListDetail encodes it for
	// compatibility, but public detail projection semantics are unverified.
	Fields string `url:"fields,omitempty"`
	ListOptions
}

// BackupSchedulersService provides public scheduler reads. Mutations and drive
// binding are separate operations; no automatic pagination or polling occurs.
// Docs: https://docs.cloudsigma.com/en/latest/backup_schedulers.html
type BackupSchedulersService service

// List returns the documented default scheduler list and pagination metadata.
func (s *BackupSchedulersService) List(ctx context.Context, opts *BackupSchedulerListOptions) ([]BackupScheduler, *Response, error) {
	return s.list(ctx, "backupschedulers/", opts)
}

// ListDetail returns scheduler objects with detailed schedule information.
func (s *BackupSchedulersService) ListDetail(ctx context.Context, opts *BackupSchedulerListOptions) ([]BackupScheduler, *Response, error) {
	return s.list(ctx, "backupschedulers/detail/", opts)
}

func (s *BackupSchedulersService) list(ctx context.Context, path string, opts *BackupSchedulerListOptions) ([]BackupScheduler, *Response, error) {
	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	var root struct {
		Objects []BackupScheduler `json:"objects"`
		Meta    *Meta             `json:"meta"`
	}
	resp, err := s.client.Do(ctx, req, &root)
	if err != nil {
		return nil, resp, err
	}
	if root.Meta != nil {
		resp.Meta = root.Meta
	}
	return root.Objects, resp, nil
}

// Get returns a single scheduler object. Conservative SDK URL safety permits
// only ASCII letters/digits/hyphen/underscore, not arbitrary URL components.
func (s *BackupSchedulersService) Get(ctx context.Context, id string) (BackupScheduler, *Response, error) {
	if id == "" {
		return nil, nil, ErrEmptyArgument
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return nil, nil, fmt.Errorf("invalid backup scheduler identifier")
		}
	}
	req, err := s.client.NewRequest(http.MethodGet, fmt.Sprintf("backupschedulers/%s/", id), nil)
	if err != nil {
		return nil, nil, err
	}
	var scheduler BackupScheduler
	resp, err := s.client.Do(ctx, req, &scheduler)
	if err != nil {
		return nil, resp, err
	}
	return scheduler, resp, nil
}
