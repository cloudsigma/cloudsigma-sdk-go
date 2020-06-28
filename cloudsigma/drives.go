package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const driveBasePath = "drives"

// DrivesService handles communication with the drives related methods of the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/drives.html
type DrivesService service

// Drive represents a CloudSigma drive.
type Drive struct {
	Media       string    `json:"media,omitempty"`
	Name        string    `json:"name,omitempty"`
	Owner       Ownership `json:"owner,omitempty"`
	ResourceURI string    `json:"resource_uri,omitempty"`
	Size        int       `json:"size,omitempty"`
	Status      string    `json:"status,omitempty"`
	StorageType string    `json:"storage_type,omitempty"`
	Tags        []Tag     `json:"tags,omitempty"`
	UUID        string    `json:"uuid"`
}

type DriveCloneRequest struct {
	Media       string `json:"media,omitempty"`
	Name        string `json:"name,omitempty"`
	Size        int    `json:"size,omitempty"`
	StorageType string `json:"storage_type,omitempty"`
}

// Get provides detailed information for drive identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/drives.html#list-single-drive
func (s *DrivesService) Get(ctx context.Context, uuid string) (*Drive, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v", driveBasePath, uuid)

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
