package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const drivesBasePath = "drives"

// DrivesService handles communication with the drives related methods of
// the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/drives.html
type DrivesService service

// Drive represents a CloudSigma drive.
type Drive struct {
	AllowMultimount bool                   `json:"allow_multimount,omitempty"`
	Licenses        []DriveLicense         `json:"licenses,omitempty"`
	Media           string                 `json:"media,omitempty"`
	Meta            map[string]interface{} `json:"meta,omitempty"`
	MountedOn       []ResourceLink         `json:"mounted_on,omitempty"`
	Name            string                 `json:"name,omitempty"`
	Owner           ResourceLink           `json:"owner,omitempty"`
	RemoteSnapshots []ResourceLink         `json:"remote_snapshots,omitempty"`
	ResourceURI     string                 `json:"resource_uri,omitempty"`
	Runtime         DriveRuntime           `json:"runtime,omitempty"`
	Size            int                    `json:"size,omitempty"`
	Snapshots       []ResourceLink         `json:"snapshots,omitempty"`
	Status          string                 `json:"status,omitempty"`
	StorageType     string                 `json:"storage_type,omitempty"`
	Tags            []Tag                  `json:"tags,omitempty"`
	UUID            string                 `json:"uuid,omitempty"`
}

// DriveRuntime represents a CloudSigma runtime information of the drive.
type DriveRuntime struct {
	IsSnapshotable         bool   `json:"is_snapshotable,omitempty"`
	SnapshotsAllocatedSize int    `json:"snapshots_allocated_size,omitempty"`
	StorageType            string `json:"storage_type,omitempty"`
}

// DriveCreateRequest represents a request to create a drive.
type DriveCreateRequest struct {
	Drives []Drive `json:"objects"`
}

// DriveUpdateRequest represents a request to update a drive.
type DriveUpdateRequest struct {
	*Drive
}

// DriveCloneRequest represents a request to clone a drive.
type DriveCloneRequest struct {
	*Drive
}

type drivesRoot struct {
	Drives []Drive `json:"objects"`
	Meta   *Meta   `json:"meta,omitempty"`
}

// List provides a detailed list of drives with additional information to which
// the authenticated user has access.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/drives.html#detailed-listing
func (s *DrivesService) List(ctx context.Context) ([]Drive, *Response, error) {
	path := fmt.Sprintf("%v/detail/", drivesBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(drivesRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.Drives, resp, nil
}

// Get provides detailed information for drive identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/drives.html#list-single-drive
func (s *DrivesService) Get(ctx context.Context, uuid string) (*Drive, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", drivesBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	drive := new(Drive)
	resp, err := s.client.Do(ctx, req, drive)
	if err != nil {
		return nil, resp, err
	}

	return drive, resp, nil
}

// Create makes a new drive (or drives) with given payload.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/drives.html#creating
func (s *DrivesService) Create(ctx context.Context, createRequest *DriveCreateRequest) ([]Drive, *Response, error) {
	if createRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/", drivesBasePath)

	req, err := s.client.NewRequest(http.MethodPost, path, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(DriveCreateRequest)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Drives, resp, nil
}

// Update edits a drive identified by uuid. Note that if the drive is mounted
// on a running server only the name, meta, tags, and allow_multimount can be
// changed.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/drives.html#editing
func (s *DrivesService) Update(ctx context.Context, uuid string, updateRequest *DriveUpdateRequest) (*Drive, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	if updateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/%v/", drivesBasePath, uuid)

	// by update UUID must be empty
	updateRequest.UUID = ""

	req, err := s.client.NewRequest(http.MethodPut, path, updateRequest)
	if err != nil {
		return nil, nil, err
	}

	drive := new(Drive)
	resp, err := s.client.Do(ctx, req, drive)
	if err != nil {
		return nil, resp, err
	}

	return drive, resp, nil
}

// Delete removes a single drive identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/drives.html#single-drive
func (s *DrivesService) Delete(ctx context.Context, uuid string) (*Response, error) {
	if uuid == "" {
		return nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", drivesBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Resize updates a drive definition. Note that the resize action is a full
// definition update (it can update even name and metadata), so a full
// definition should be provided to this call.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/drives.html#resizing-update-or-fail
func (s *DrivesService) Resize(ctx context.Context, uuid string, updateRequest *DriveUpdateRequest) ([]Drive, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	if updateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/%v/action/?do=resize", drivesBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodPost, path, updateRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(drivesRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Drives, resp, nil
}

// Clone duplicates a drive. DriveCloneRequest is optional. Size of the
// cloned drive can only be bigger or the same.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/drives.html#cloning
func (s *DrivesService) Clone(ctx context.Context, uuid string, cloneRequest *DriveCloneRequest) (*Drive, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/action/?do=clone", drivesBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, nil, err
	}
	if cloneRequest != nil {
		req, err = s.client.NewRequest(http.MethodPost, path, cloneRequest)
		if err != nil {
			return nil, nil, err
		}
	}

	root := new(drivesRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return &root.Drives[0], resp, nil
}
