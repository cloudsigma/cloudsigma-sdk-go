package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const capabilitiesBasePath = "capabilities"

// CapabilitiesService handles communication with the capabilities related methods of the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/capabilities.html
type CapabilitiesService service

// Capabilities represents CloudSigma capabilities.
type Capabilities struct {
	Hosts       CapabilitiesHosts       `json:"hosts,omitempty"`
	Hypervisors CapabilitiesHypervisors `json:"hypervisors,omitempty"`
}

// CapabilitiesHosts represents available host types and their limitations.
type CapabilitiesHosts struct {
	AMD   CapabilitiesHost `json:"amd,omitempty"`
	Intel CapabilitiesHost `json:"intel,omitempty"`
}

// CapabilitiesHost represents capabilities limitation for an host type
type CapabilitiesHost struct {
	CPU       CapabilitiesLimitation `json:"cpu,omitempty"`
	CPUPerSMP CapabilitiesLimitation `json:"cpu_per_smp,omitempty"`
	Memory    CapabilitiesLimitation `json:"mem,omitempty"`
	SMP       CapabilitiesLimitation `json:"smp,omitempty"`
}

// CapabilitiesHypervisors represents a list of hypervisors and which hosts they are available on.
type CapabilitiesHypervisors struct {
	KVM []string `json:"kvm,omitempty"`
}

// CapabilitiesLimitation represents capabilities limitation.
type CapabilitiesLimitation struct {
	Max int `json:"max,omitempty"`
	Min int `json:"min,omitempty"`
}

// Get provides the capabilities object.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/capabilities.html#capabilities
func (s *CapabilitiesService) Get(ctx context.Context) (*Capabilities, *Response, error) {
	path := fmt.Sprintf("%v/", capabilitiesBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	capabilities := new(Capabilities)
	resp, err := s.client.Do(ctx, req, capabilities)
	if err != nil {
		return nil, resp, err
	}

	return capabilities, resp, nil
}
