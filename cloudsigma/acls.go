package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const aclsBasePath = "acls"

type ACLsService service

type ACL struct {
	Name        string    `json:"name,omitempty"`
	Owner       Ownership `json:"owner,omitempty"`
	ResourceURI string    `json:"resource_uri,omitempty"`
	Rules       []Rule    `json:"rules,omitempty"`
	UUID        string    `json:"uuid"`
}

type Rule struct {
	Permission string `json:"permission,omitempty"`
}

type ACLCreateRequest struct {
	ACLs []ACL `json:"objects"`
}

type ACLUpdateRequest struct {
	*ACL
}

type aclsRoot struct {
	Meta *Meta `json:"meta,omitempty"`
	ACLs []ACL `json:"objects"`
}

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

func (s *ACLsService) Get(ctx context.Context, uuid string) (*ACL, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v", aclsBasePath, uuid)

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

func (s *ACLsService) Create(ctx context.Context, aclCreateRequest *ACLCreateRequest) ([]ACL, *Response, error) {
	if aclCreateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/", aclsBasePath)

	req, err := s.client.NewRequest(http.MethodPost, path, aclCreateRequest)
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

func (s *ACLsService) Update(ctx context.Context, uuid string, aclUpdateRequest *ACLUpdateRequest) (*ACL, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	if aclUpdateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/%v/", aclsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodPut, path, aclUpdateRequest)
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
