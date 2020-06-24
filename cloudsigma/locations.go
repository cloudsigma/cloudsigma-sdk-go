package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const locationsBasePath = "locations"

// LocationsService handles communication with the location related methods of the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/locations.html
type LocationsService service

// Location represents a CloudSigma location.
type Location struct {
	AlternativeFrontendURL   string `json:"alternative_frontend_url"`
	APIEndpoint              string `json:"api_endpoint"`
	CountryCode              string `json:"country_code"`
	DefaultFrontendSignupURL string `json:"default_frontend_signup_url"`
	DefaultFrontendURL       string `json:"default_frontend_url"`
	DisplayName              string `json:"display_name"`
	DocumentationURL         string `json:"documentation_url"`
	ID                       string `json:"id"`
	UploadURL                string `json:"upload_url"`
	WebsocketURL             string `json:"websocket_url"`
}

type locationsRoot struct {
	Locations []Location `json:"objects"`
}

// List provides a list of the currently available CloudSigma locations, and information on specific urls,
// such as the websockets and upload urls.
func (s *LocationsService) List(ctx context.Context) ([]Location, *http.Response, error) {
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

	return root.Locations, resp, err
}
