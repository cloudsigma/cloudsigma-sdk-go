package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const ipsBasePath = "ips"

// IPsService handles communication with the IPs related methods of
// the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/networking.html#ips
type IPsService service

// IP represents a CloudSigma IP address.
type IP struct {
	Gateway     string                 `json:"gateway,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	Nameservers []string               `json:"nameservers,omitempty"`
	Netmask     int                    `json:"netmask,omitempty"`
	Owner       Ownership              `json:"owner,omitempty"`
	ResourceURI string                 `json:"resource_uri,omitempty"`
	Server      Ownership              `json:"server,omitempty"`
	Tags        []Tag                  `json:"tags,omitempty"`
	UUID        string                 `json:"uuid"`
}

type ipsRoot struct {
	Meta *Meta `json:"meta,omitempty"`
	IPs  []IP  `json:"objects"`
}

// List provides a list of IPs to which the authenticated user has access.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/networking.html#id2
func (s *IPsService) List(ctx context.Context) ([]IP, *Response, error) {
	path := fmt.Sprintf("%v/detail/", ipsBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(ipsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.IPs, resp, nil
}

// Get provides detailed information for IP address identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/networking.html#get-single-ip
func (s *IPsService) Get(ctx context.Context, uuid string) (*IP, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", ipsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	ip := new(IP)
	resp, err := s.client.Do(ctx, req, ip)
	if err != nil {
		return nil, resp, err
	}

	return ip, resp, nil
}
