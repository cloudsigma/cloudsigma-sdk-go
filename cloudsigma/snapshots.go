package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const snapshotsBasePath = "snapshots"

// SnapshotsService handles communication with the snapshot related methods of
// the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/snapshots.html
type SnapshotsService service

// Snapshot represents a CloudSigma snapshot.
type Snapshot struct {
	AllocatedSize int                    `json:"allocated_size,omitempty"`
	Drive         Drive                  `json:"drive,omitempty"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
	Name          string                 `json:"name,omitempty"`
	Owner         ResourceLink           `json:"owner,omitempty"`
	ResourceURI   string                 `json:"resource_uri,omitempty"`
	Status        string                 `json:"status,omitempty"`
	Tags          []Tag                  `json:"tags,omitempty"`
	Timestamp     string                 `json:"timestamp,omitempty"`
	UUID          string                 `json:"uuid,omitempty"`
}

// SnapshotCreateRequest represents a request to create a snapshot.
type SnapshotCreateRequest struct {
	Snapshots []Snapshot `json:"objects"`
}

// SnapshotUpdateRequest represents a request to update a snapshot.
type SnapshotUpdateRequest struct {
	*Snapshot
}

type snapshotsRoot struct {
	Meta      *Meta      `json:"meta,omitempty"`
	Snapshots []Snapshot `json:"objects"`
}

// List provides a detailed list of snapshots to which the authenticated user
// has access.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/snapshots.html#detailed-listing
func (s *SnapshotsService) List(ctx context.Context) ([]Snapshot, *Response, error) {
	path := fmt.Sprintf("%v/detail/", snapshotsBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(snapshotsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.Snapshots, resp, nil
}

// Get provides detailed information for snapshot identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/snapshots.html#list-single-snapshot
func (s *SnapshotsService) Get(ctx context.Context, uuid string) (*Snapshot, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", snapshotsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	snapshot := new(Snapshot)
	resp, err := s.client.Do(ctx, req, snapshot)
	if err != nil {
		return nil, resp, err
	}

	return snapshot, resp, nil
}

// Create makes a new snapshot with given payload.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/snapshots.html#creating
func (s *SnapshotsService) Create(ctx context.Context, createRequest *SnapshotCreateRequest) ([]Snapshot, *Response, error) {
	if createRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/", snapshotsBasePath)

	req, err := s.client.NewRequest(http.MethodPost, path, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(SnapshotCreateRequest)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Snapshots, resp, nil
}

// Update edits a snapshot identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/snapshots.html#editing
func (s *SnapshotsService) Update(ctx context.Context, uuid string, updateRequest *SnapshotUpdateRequest) (*Snapshot, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	if updateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/%v/", snapshotsBasePath, uuid)

	// by update UUID must be empty
	updateRequest.UUID = ""

	req, err := s.client.NewRequest(http.MethodPut, path, updateRequest)
	if err != nil {
		return nil, nil, err
	}

	snapshot := new(Snapshot)
	resp, err := s.client.Do(ctx, req, snapshot)
	if err != nil {
		return nil, resp, err
	}

	return snapshot, resp, nil
}

// Delete removes a snapshot identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/snapshots.html#deleting
func (s *SnapshotsService) Delete(ctx context.Context, uuid string) (*Response, error) {
	if uuid == "" {
		return nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", snapshotsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
