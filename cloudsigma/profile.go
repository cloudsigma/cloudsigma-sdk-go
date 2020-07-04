package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const profileBasePath = "profile"

// ProfileService handles communication with the profile related methods of the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/profile.html
type ProfileService service

type Profile struct {
	Address   string `json:"address,omitempty"`
	Company   string `json:"company,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Title     string `json:"title,omitempty"`
	UUID      string `json:"uuid"`
}

// ProfileUpdateRequest represents a request to update an ACL.
type ProfileUpdateRequest struct {
	*Profile
}

// Get provides information for an user profile.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/profile.html#listing
func (s *ProfileService) Get(ctx context.Context) (*Profile, *Response, error) {
	path := fmt.Sprintf("%v/", profileBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	profile := new(Profile)
	resp, err := s.client.Do(ctx, req, profile)
	if err != nil {
		return nil, resp, err
	}

	return profile, resp, nil
}

// Update edits a user profile.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/profile.html#editing
func (s *ProfileService) Update(ctx context.Context, profileUpdateRequest *ProfileUpdateRequest) (*Profile, *Response, error) {
	if profileUpdateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/", profileBasePath)

	req, err := s.client.NewRequest(http.MethodPut, path, profileUpdateRequest)
	if err != nil {
		return nil, nil, err
	}

	profile := new(Profile)
	resp, err := s.client.Do(ctx, req, profile)
	if err != nil {
		return nil, resp, err
	}

	return profile, resp, nil
}
