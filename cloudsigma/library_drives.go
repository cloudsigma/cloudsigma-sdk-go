package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const libdriveBasePath = "libdrives"

// LibraryDrivesService handles communication with the library drives related
// methods of the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/libdrives.html
type LibraryDrivesService service

// LibraryDrive represents a CloudSigma library drive.
type LibraryDrive struct {
	Arch        string                 `json:"arch,omitempty"`
	Description string                 `json:"description,omitempty"`
	Favourite   bool                   `json:"favourite,omitempty"`
	ImageType   string                 `json:"image_type,omitempty"`
	Media       string                 `json:"media,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	Name        string                 `json:"name,omitempty"`
	OS          string                 `json:"os,omitempty"`
	Paid        bool                   `json:"paid,omitempty"`
	ResourceURI string                 `json:"resource_uri,omitempty"`
	Size        int                    `json:"size,omitempty"`
	Status      string                 `json:"status,omitempty"`
	StorageType string                 `json:"storage_type,omitempty"`
	UUID        string                 `json:"uuid"`
	Version     string                 `json:"version,omitempty"`
}

// LibraryDriveCloneRequest represents a request to clone a library drive.
type LibraryDriveCloneRequest struct {
	*LibraryDrive
}

type libraryDrivesRoot struct {
	LibraryDrives []LibraryDrive `json:"objects"`
	Meta          *Meta          `json:"meta,omitempty"`
}

// List provides a list of library drives to which the authenticated user has access.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/libdrives.html#listing
func (s *LibraryDrivesService) List(ctx context.Context) ([]LibraryDrive, *Response, error) {
	path := fmt.Sprintf("%v/", libdriveBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(libraryDrivesRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.LibraryDrives, resp, nil
}

// Get provides detailed information for library drive identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/libdrives.html#list-single-drive
func (s *LibraryDrivesService) Get(ctx context.Context, uuid string) (*LibraryDrive, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v", libdriveBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	libdrive := new(LibraryDrive)
	resp, err := s.client.Do(ctx, req, libdrive)
	if err != nil {
		return nil, resp, err
	}

	return libdrive, resp, nil
}

// Clone duplicates a drive. LibraryDriveCloneRequest is optional. Size of the
// cloned drive can only be bigger or the same.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/libdrives.html#cloning-library-drive
func (s *LibraryDrivesService) Clone(ctx context.Context, uuid string, cloneRequest *LibraryDriveCloneRequest) (*LibraryDrive, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/action/?do=clone", libdriveBasePath, uuid)

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

	root := new(libraryDrivesRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return &root.LibraryDrives[0], resp, nil
}
