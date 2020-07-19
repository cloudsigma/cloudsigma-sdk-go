package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const vlansBasePath = "vlans"

// VLANsService handles communication with the VLAN related methods of the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/networking.html#vlan
type VLANsService service

// VLAN represents a CloudSigma VLAN.
type VLAN struct {
	Meta         map[string]interface{} `json:"meta,omitempty"`
	Owner        *ResourceLink          `json:"owner,omitempty"`
	ResourceURI  string                 `json:"resource_uri,omitempty"`
	Subscription *VLANSubscription      `json:"subscription,omitempty"`
	UUID         string                 `json:"uuid,omitempty"`
}

// VLANSubscription represents a CloudSigma subscription reference.
type VLANSubscription struct {
	ID          int    `json:"id"`
	ResourceURI string `json:"resource_uri,omitempty"`
}

// VLANUpdateRequest represents a request to update a VLAN.
type VLANUpdateRequest struct {
	*VLAN
}

type vlansRoot struct {
	Meta  *Meta  `json:"meta,omitempty"`
	VLANs []VLAN `json:"objects"`
}

// List provides a list of VLANs to which the authenticated user has access.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/networking.html#detailed-listing
func (s *VLANsService) List(ctx context.Context) ([]VLAN, *Response, error) {
	path := fmt.Sprintf("%v/detail/", vlansBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(vlansRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.VLANs, resp, nil
}

// Get provides detailed information for VLAN identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/networking.html#get-single-vlan
func (s *VLANsService) Get(ctx context.Context, uuid string) (*VLAN, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", vlansBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	vlan := new(VLAN)
	resp, err := s.client.Do(ctx, req, vlan)
	if err != nil {
		return nil, resp, err
	}

	return vlan, resp, nil
}

// Update edits a VLAN identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/networking.html#editing
func (s *VLANsService) Update(ctx context.Context, uuid string, vlanUpdateRequest *VLANUpdateRequest) (*VLAN, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	if vlanUpdateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/%v/", vlansBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodPut, path, vlanUpdateRequest)
	if err != nil {
		return nil, nil, err
	}

	vlan := new(VLAN)
	resp, err := s.client.Do(ctx, req, vlan)
	if err != nil {
		return nil, resp, err
	}

	return vlan, resp, nil
}
