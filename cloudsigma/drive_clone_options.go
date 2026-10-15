package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// DriveCloneOptions provides opt-in presence-aware clone fields without changing
// DriveCloneRequest. Definition contributes the legacy Drive JSON fields.
// A nil AllowMultimount retains Definition's legacy encoding (false omitted).
// A nonnil AllowMultimount overrides Definition and sends true or false explicitly.
// Neither the definition nor the override is mutated.
type DriveCloneOptions struct {
	Definition      *Drive
	AllowMultimount *bool
}

// driveCloneBody shadows only the private wire representation's boolean.
type driveCloneBody struct {
	*Drive
	AllowMultimount *bool `json:"allow_multimount,omitempty"`
}

// CloneWithOptions duplicates a drive with presence-aware optional fields.
// Nil options omit the body; empty options send {}. A 202 response acknowledges
// acceptance only: callers must observe the drive separately for completion.
// Size of the cloned drive can only be bigger or the same.
//
// CloudSigma API docs: https://docs.cloudsigma.com/en/latest/drives.html#cloning
func (s *DrivesService) CloneWithOptions(ctx context.Context, uuid string, options *DriveCloneOptions) (*Drive, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/action/?do=clone", drivesBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, nil, err
	}
	if options != nil {
		body := &driveCloneBody{Drive: options.Definition, AllowMultimount: options.AllowMultimount}
		if body.AllowMultimount == nil && options.Definition != nil && options.Definition.AllowMultimount {
			value := true
			body.AllowMultimount = &value
		}
		req, err = s.client.NewRequest(http.MethodPost, path, body)
		if err != nil {
			return nil, nil, err
		}
	}

	root := new(drivesRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	if len(root.Drives) == 0 {
		return nil, resp, fmt.Errorf("clone response contains no drives")
	}

	return (*Drive)(&root.Drives[0]), resp, nil
}
