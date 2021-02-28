package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const remoteSnapshotsBasePath = "remotesnapshots"

// RemoteSnapshotsService handles communication with the remote snapshot
// related methods of the CloudSigma API.
type RemoteSnapshotsService service

// RemoteSnapshot represents a CloudSigma remote snapshot.
type RemoteSnapshot struct {
	Location                    string                       `json:"location,omitempty"`
	RemoteSnapshotDriveMetadata *RemoteSnapshotDriveMetadata `json:"drive_meta,omitempty"`
	Snapshot
}

// RemoteSnapshotDriveMetadata represents a CloudSigma snapshot drive meta.
type RemoteSnapshotDriveMetadata struct {
	Media       string `json:"media,omitempty"`
	Name        string `json:"name,omitempty"`
	Size        int    `json:"size,omitempty"`
	SourceUUID  string `json:"src_uuid,omitempty"`
	StorageType string `json:"storage_type,omitempty"`
}

// RemoteSnapshotCreateRequest represents a request to create a remote snapshot.
type RemoteSnapshotCreateRequest struct {
	RemoteSnapshots []RemoteSnapshot `json:"objects"`
}

// RemoteSnapshotUpdateRequest represents a request to update a remote snapshot.
type RemoteSnapshotUpdateRequest struct {
	*RemoteSnapshot
}

type remoteSnapshotsRoot struct {
	RemoteSnapshots []RemoteSnapshot `json:"objects"`
	Meta            *Meta            `json:"meta,omitempty"`
}

func (r RemoteSnapshot) String() string {
	return Stringify(r)
}

// List provides a detailed list of remote snapshots to which the authenticated
// user has access.
func (s *RemoteSnapshotsService) List(ctx context.Context, opts *ListOptions) ([]RemoteSnapshot, *Response, error) {
	path := fmt.Sprintf("%v/detail/", remoteSnapshotsBasePath)
	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(remoteSnapshotsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.RemoteSnapshots, resp, nil
}

// Get provides detail information for remote snapshot identified by uuid.
func (s *RemoteSnapshotsService) Get(ctx context.Context, uuid string) (*RemoteSnapshot, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", remoteSnapshotsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	remoteSnapshot := new(RemoteSnapshot)
	resp, err := s.client.Do(ctx, req, remoteSnapshot)
	if err != nil {
		return nil, resp, err
	}

	return remoteSnapshot, resp, err
}

// Create makes a new remote snapshot with given payload.
func (s *RemoteSnapshotsService) Create(ctx context.Context, createRequest *RemoteSnapshotCreateRequest) ([]RemoteSnapshot, *Response, error) {
	if createRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/", remoteSnapshotsBasePath)

	req, err := s.client.NewRequest(http.MethodPost, path, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(remoteSnapshotsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.RemoteSnapshots, resp, nil
}

// Update edits a remote snapshot identified by uuid.
func (s *RemoteSnapshotsService) Update(ctx context.Context, uuid string, updateRequest *RemoteSnapshotUpdateRequest) (*RemoteSnapshot, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	if updateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/%v/", remoteSnapshotsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodPut, path, updateRequest)
	if err != nil {
		return nil, nil, err
	}

	remoteSnapshot := new(RemoteSnapshot)
	resp, err := s.client.Do(ctx, req, remoteSnapshot)
	if err != nil {
		return nil, resp, err
	}

	return remoteSnapshot, resp, nil
}

// Delete removes a remote snapshot identified by uuid.
func (s *RemoteSnapshotsService) Delete(ctx context.Context, uuid string) (*Response, error) {
	if uuid == "" {
		return nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", remoteSnapshotsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
