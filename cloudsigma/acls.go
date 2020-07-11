package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const aclsBasePath = "acls"

// ACLsService handles communication with the ACL (Access Control Lists) related methods of the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/acls.html
type ACLsService service

// ACL represents a CloudSigma ACL.
type ACL struct {
	Meta        map[string]interface{} `json:"meta,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Owner       Ownership              `json:"owner,omitempty"`
	ResourceURI string                 `json:"resource_uri,omitempty"`
	Rules       []ACLRule              `json:"rules,omitempty"`
	Tags        []Tag                  `json:"tags,omitempty"`
	UUID        string                 `json:"uuid"`
}

// ACLRule represents a CloudSigma ACL rule.
type ACLRule struct {
	Permission string `json:"permission,omitempty"`
}

// ACLCreateRequest represents a request to create an ACL.
type ACLCreateRequest struct {
	ACLs []ACL `json:"objects"`
}

// ACLUpdateRequest represents a request to update an ACL.
type ACLUpdateRequest struct {
	*ACL
}

type aclsRoot struct {
	ACLs []ACL `json:"objects"`
	Meta *Meta `json:"meta,omitempty"`
}

// List provides a list of ACLs defined by the authenticated user.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/acls.html#listing
func (s *ACLsService) List(ctx context.Context) ([]ACL, *Response, error) {
	path := fmt.Sprintf("%v/", aclsBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(aclsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.ACLs, resp, nil
}

// Get provides detailed information for an ACL identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/acls.html#list-single-acl
func (s *ACLsService) Get(ctx context.Context, uuid string) (*ACL, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", aclsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	acl := new(ACL)
	resp, err := s.client.Do(ctx, req, acl)
	if err != nil {
		return nil, resp, err
	}

	return acl, resp, nil
}

// Create makes a new ACL (or ACLs) with given payload.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/acls.html#creating
func (s *ACLsService) Create(ctx context.Context, createRequest *ACLCreateRequest) ([]ACL, *Response, error) {
	if createRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/", aclsBasePath)

	req, err := s.client.NewRequest(http.MethodPost, path, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(ACLCreateRequest)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.ACLs, resp, nil
}

// Update edits an ACL identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/acls.html#editing
func (s *ACLsService) Update(ctx context.Context, uuid string, updateRequest *ACLUpdateRequest) (*ACL, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	if updateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/%v/", aclsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodPut, path, updateRequest)
	if err != nil {
		return nil, nil, err
	}

	acl := new(ACL)
	resp, err := s.client.Do(ctx, req, acl)
	if err != nil {
		return nil, resp, err
	}

	return acl, resp, nil
}

// Delete removes a single ACL identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/acls.html#deleting
func (s *ACLsService) Delete(ctx context.Context, uuid string) (*Response, error) {
	if uuid == "" {
		return nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", aclsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
