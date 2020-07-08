package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const cloudStatusBasePath = "cloud_status"

// CloudStatusService handles communication with the cloud status related methods of the CloudSigma API.
type CloudStatusService service

// CloudStatus represents a CloudSigma cloud status.
type CloudStatus struct {
	FreeTier              CloudStatusFreeTier        `json:"free_tier,omitempty"`
	FreeTierMonthly       CloudStatusFreeTierMonthly `json:"free_tier_monthly,omitempty"`
	Guest                 bool                       `json:"guest,omitempty"`
	HostAvailabilityZones bool                       `json:"host_availability_zones,omitempty"`
	RemoteSnapshots       bool                       `json:"remote_snapshots,omitempty"`
	Signup                bool                       `json:"signup,omitempty"`
	SSO                   []string                   `json:"sso,omitempty"`
	Trial                 bool                       `json:"trial,omitempty"`
	VMware                bool                       `json:"vmware,omitempty"`
	VPC                   bool                       `json:"vpc,omitempty"`
}

// CloudStatusFreeTier represents a CloudSigma cloud status free tier.
type CloudStatusFreeTier struct {
	DSSD   int `json:"dssd,omitempty"`
	Memory int `json:"mem,omitempty"`
}

// CloudStatusFreeTierMonthly represents a CloudSigma cloud status monthly free tier.
type CloudStatusFreeTierMonthly struct {
	TX int `json:"tx,omitempty"`
}

// Get provides  information for cloud status.
func (s *CloudStatusService) Get(ctx context.Context) (*CloudStatus, *Response, error) {
	path := fmt.Sprintf("%v/", cloudStatusBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	cloudStatus := new(CloudStatus)
	resp, err := s.client.Do(ctx, req, cloudStatus)
	if err != nil {
		return nil, resp, err
	}

	return cloudStatus, resp, nil
}
