package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const fwpoliciesBasePath = "fwpolicies"

// FirewallPoliciesService handles communication with the firewall policies
// related methods of the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/fwpolicies.html
type FirewallPoliciesService service

// FirewallPolicy represents a CloudSigma firewall policy.
type FirewallPolicy struct {
	Meta        map[string]interface{} `json:"meta,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Owner       Ownership              `json:"owner,omitempty"`
	ResourceURI string                 `json:"resource_uri,omitempty"`
	Rules       []FirewallPolicyRule   `json:"rules,omitempty"`
	Servers     []Ownership            `json:"servers,omitempty"`
	Tags        []Tag                  `json:"tags,omitempty"`
	UUID        string                 `json:"uuid,omitempty"`
}

// FirewallPolicyRule represents a CloudSigma firewall policy rule.
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

// FirewallPolicyCreateRequest represents a request to create a firewall policy.
type FirewallPolicyCreateRequest struct {
	FirewallPolicies []FirewallPolicy `json:"objects,omitempty"`
}

// FirewallPolicyUpdateRequest represents a request to update a firewall policy.
type FirewallPolicyUpdateRequest struct {
	*FirewallPolicy
}

type fwpoliciesRoot struct {
	Meta             *Meta            `json:"meta,omitempty"`
	FirewallPolicies []FirewallPolicy `json:"objects"`
}

// List provides a detailed list of firewall policies to which the authenticated
// user has access.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/fwpolicies.html#detailed-listing
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

// Get provides detailed information for a firewall policy identified by uuid.
//
// CloudSigma API docs:
func (s *FirewallPoliciesService) Get(ctx context.Context, uuid string) (*FirewallPolicy, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", fwpoliciesBasePath, uuid)

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

// Create makes a new firewall policy (or policies) with given payload.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/fwpolicies.html#create
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

// Update edits an existing firewall policy.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/fwpolicies.html#editing
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

// Delete removes a firewall policy identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/fwpolicies.html#delete
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
