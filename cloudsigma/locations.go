package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const locationsBasePath = "locations"

// LocationsService handles communication with the location related methods of
// the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/locations.html
type LocationsService service

// Location represents a CloudSigma location.
type Location struct {
	AlternativeFrontendURL   string `json:"alternative_frontend_url,omitempty"`
	APIEndpoint              string `json:"api_endpoint,omitempty"`
	CountryCode              string `json:"country_code,omitempty"`
	DefaultFrontendSignupURL string `json:"default_frontend_signup_url,omitempty"`
	DefaultFrontendURL       string `json:"default_frontend_url,omitempty"`
	DisplayName              string `json:"display_name,omitempty"`
	DocumentationURL         string `json:"documentation_url,omitempty"`
	ID                       string `json:"id,omitempty"`
	UploadURL                string `json:"upload_url,omitempty"`
	WebsocketURL             string `json:"websocket_url,omitempty"`
}

type locationsRoot struct {
	Locations []Location `json:"objects"`
	Meta      *Meta      `json:"meta,omitempty"`
}

// List provides a list of the currently available CloudSigma locations,
// and information on specific urls, such as the websockets and upload urls.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/locations.html#locations
func (s *LocationsService) List(ctx context.Context) ([]Location, *Response, error) {
	path := fmt.Sprintf("%v/", locationsBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(locationsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.Locations, resp, nil
}
