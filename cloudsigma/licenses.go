package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const licensesBasePath = "licenses"

type LicensesService service

type License struct {
	Burstable   bool   `json:"burstable,omitempty"`
	LongName    string `json:"long_name,omitempty"`
	Name        string `json:"name,omitempty"`
	ResourceURI string `json:"resource_uri,omitempty"`
	Type        string `json:"type,omitempty"`
	UserMetric  string `json:"user_metric"`
}

type licensesRoot struct {
	Meta     *Meta     `json:"meta,omitempty"`
	Licenses []License `json:"objects"`
}

func (s *LicensesService) List(ctx context.Context) ([]License, *Response, error) {
	path := fmt.Sprintf("%v/", licensesBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(licensesRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.Licenses, resp, nil
}
