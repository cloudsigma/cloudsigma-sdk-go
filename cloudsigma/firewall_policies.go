package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const fwpoliciesBasePath = "fwpolicies"

type FirewallPoliciesService service

type FirewallPolicy struct {
	Name        string               `json:"name,omitempty"`
	Owner       Ownership            `json:"owner,omitempty"`
	ResourceURI string               `json:"resource_uri,omitempty"`
	Rules       []FirewallPolicyRule `json:"rules,omitempty"`
	UUID        string               `json:"uuid,omitempty"`
}

type FirewallPolicyRule struct {
	Action          string `json:"action,omitempty"`
	Comment         string `json:"comment,omitempty"`
	Direction       string `json:"direction,omitempty"`
	DestinationIP   string `json:"dst_ip,omitempty"`
	DestinationPort string `json:"dst_port,omitempty"`
	Protocol        string `json:"ip_proto,omitempty"`
	SourceIP        string `json:"src_ip,omitempty"`
	SourcePort      string `json:"src_port,omitempty"`
}

type FirewallPolicyCreateRequest struct {
	FirewallPolicies []FirewallPolicy `json:"objects,omitempty"`
}

type FirewallPolicyUpdateRequest struct {
	*FirewallPolicy
}

type fwpoliciesRoot struct {
	Meta             *Meta            `json:"meta,omitempty"`
	FirewallPolicies []FirewallPolicy `json:"objects"`
}

func (s *FirewallPoliciesService) List(ctx context.Context) ([]FirewallPolicy, *Response, error) {
	path := fmt.Sprintf("%v/detail/", fwpoliciesBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(fwpoliciesRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.FirewallPolicies, resp, nil
}

func (s *FirewallPoliciesService) Get(ctx context.Context, uuid string) (*FirewallPolicy, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v", fwpoliciesBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	fwpolicy := new(FirewallPolicy)
	resp, err := s.client.Do(ctx, req, fwpolicy)
	if err != nil {
		return nil, resp, err
	}

	return fwpolicy, resp, nil
}

func (s *FirewallPoliciesService) Create(ctx context.Context, createRequest *FirewallPolicyCreateRequest) ([]FirewallPolicy, *Response, error) {
	if createRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/", fwpoliciesBasePath)

	req, err := s.client.NewRequest(http.MethodPost, path, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(FirewallPolicyCreateRequest)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.FirewallPolicies, resp, nil
}

func (s *FirewallPoliciesService) Update(ctx context.Context, uuid string, updateRequest *FirewallPolicyUpdateRequest) (*FirewallPolicy, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	if updateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/%v/", fwpoliciesBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodPut, path, updateRequest)
	if err != nil {
		return nil, nil, err
	}

	fwpolicy := new(FirewallPolicy)
	resp, err := s.client.Do(ctx, req, fwpolicy)
	if err != nil {
		return nil, resp, err
	}

	return fwpolicy, resp, nil
}

func (s *FirewallPoliciesService) Delete(ctx context.Context, uuid string) (*Response, error) {
	if uuid == "" {
		return nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", fwpoliciesBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
