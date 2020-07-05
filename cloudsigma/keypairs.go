package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const keypairsBasePath = "keypairs"

// KeypairsService handles communication with the keypairs (SSH keys) related methods of the Cloudsigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/keypairs.html
type KeypairsService service

// Keypair represents a CloudSigma keypair (ssh keys).
type Keypair struct {
	Fingerprint string `json:"fingerprint,omitempty"`
	Name        string `json:"name,omitempty"`
	PrivateKey  string `json:"private_key,omitempty"`
	PublicKey   string `json:"public_key,omitempty"`
	ResourceURI string `json:"resource_key,omitempty"`
	UUID        string `json:"uuid,omitempty"`
}

// KeypairCreateRequest represents a request to create a keypair.
type KeypairCreateRequest struct {
	Keypairs []Keypair `json:"objects"`
}

// KeypairUpdateRequest represents a request to update a keypair.
type KeypairUpdateRequest struct {
	*Keypair
}

type keypairsRoot struct {
	Keypairs []Keypair `json:"objects"`
	Meta     *Meta     `json:"meta,omitempty"`
}

// List provides a list of keypairs.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/keypairs.html#listing-getting-updating-deleting
func (s *KeypairsService) List(ctx context.Context) ([]Keypair, *Response, error) {
	path := fmt.Sprintf("%v/", keypairsBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(keypairsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.Keypairs, resp, nil
}

// Get provides information for keypair identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/keypairs.html#listing-getting-updating-deleting
func (s *KeypairsService) Get(ctx context.Context, uuid string) (*Keypair, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v", keypairsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	keypair := new(Keypair)
	resp, err := s.client.Do(ctx, req, keypair)
	if err != nil {
		return nil, resp, err
	}

	return keypair, resp, nil
}

// Create makes a new keypair (or keypairs) with given payload.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/keypairs.html#creating-a-keypair
func (s *KeypairsService) Create(ctx context.Context, keypairCreateRequest *KeypairCreateRequest) ([]Keypair, *Response, error) {
	if keypairCreateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/", keypairsBasePath)

	req, err := s.client.NewRequest(http.MethodPost, path, keypairCreateRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(KeypairCreateRequest)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Keypairs, resp, nil
}

// Update edits a keypair identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/keypairs.html#listing-getting-updating-deleting
func (s *KeypairsService) Update(ctx context.Context, uuid string, keypairUpdateRequest *KeypairUpdateRequest) (*Keypair, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	if keypairUpdateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/%v/", keypairsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodPut, path, keypairUpdateRequest)
	if err != nil {
		return nil, nil, err
	}

	keypair := new(Keypair)
	resp, err := s.client.Do(ctx, req, keypair)
	if err != nil {
		return nil, resp, err
	}

	return keypair, resp, nil
}

// Delete removes a keypair identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/keypairs.html#listing-getting-updating-deleting
func (s *KeypairsService) Delete(ctx context.Context, uuid string) (*Response, error) {
	if uuid == "" {
		return nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", keypairsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
