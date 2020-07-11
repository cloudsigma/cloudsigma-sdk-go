package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const pubkeysBasePath = "pubkeys"

// PubkeysService handles communication with the pubkeys (SSH keys) related
// methods of the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/keypairs.html#pubkeys-resource
type PubkeysService service

// List provides a list of keypairs.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/keypairs.html#pubkeys-resource
func (s *PubkeysService) List(ctx context.Context) ([]Keypair, *Response, error) {
	path := fmt.Sprintf("%v/", pubkeysBasePath)

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
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/keypairs.html#pubkeys-resource
func (s *PubkeysService) Get(ctx context.Context, uuid string) (*Keypair, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", pubkeysBasePath, uuid)

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
