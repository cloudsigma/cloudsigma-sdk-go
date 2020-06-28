package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultLocation  = "zrh"
	defaultUserAgent = "cloudsigma-sdk-go"

	baseURL         = "https://%s.cloudsigma.com/api/2.0/"
	headerRequestID = "X-REQUEST-ID"
	mediaType       = "application/json"
)

// A Client manages communication with the CloudSigma API.
type Client struct {
	client *http.Client // HTTP client used to communicate with the API.

	APIEndpoint *url.URL // Endpoint for API requests. APIEndpoint should always be specified with a trailing slash.
	UserAgent   string   // User agent used when communicating with the CloudSigma API.

	Username string // Username for CloudSigma API (user email).
	Password string // Password for CloudSigma API.

	common service // Reuse a single struct instead of allocating one for each service on the heap.

	// Drives        *DrivesService
	// IPs           *IPsService
	Keypairs *KeypairsService
	// LibraryDrives *LibraryDrivesService
	Locations *LocationsService
	// Servers       *ServersService
	Tags *TagsService
}

type service struct {
	client *Client
}

// Response is a CloudSigma response. This wraps the standard http.Response.
type Response struct {
	*http.Response

	Meta *Meta // Meta describes generic information about the response.

	RequestID string // RequestID returned from the API, useful to contact support.
}

// NewBasicAuthClient returns a new CloudSigma API client. To use API methods provide username (your email)
// and password.
func NewBasicAuthClient(username, password string) *Client {
	httpClient := &http.Client{
		Timeout: time.Second * 15,
	}

	c := &Client{
		client:    httpClient,
		UserAgent: defaultUserAgent,
		Username:  username,
		Password:  password,
	}
	c.SetLocation(defaultLocation)
	c.common.client = c

	// c.Drives = (*DrivesService)(&c.common)
	// c.IPs = (*IPsService)(&c.common)
	c.Keypairs = (*KeypairsService)(&c.common)
	// c.LibraryDrives = (*LibraryDrivesService)(&c.common)
	c.Locations = (*LocationsService)(&c.common)
	// c.Servers = (*ServersService)(&c.common)
	c.Tags = (*TagsService)(&c.common)

	return c
}

// SetLocation configures location (a sub-domain) for API endpoint.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/general.html#api-endpoint.
func (c *Client) SetLocation(location string) {
	apiEndpointUrl, _ := url.Parse(fmt.Sprintf(baseURL, location))
	c.APIEndpoint = apiEndpointUrl
}

// SetUserAgent overrides the default UserAgent.
func (c *Client) SetUserAgent(ua string) {
	c.UserAgent = ua
}

// NewRequest creates an API request. A relative URL can be provided in urlStr, in which case it is resolved
// relative to the APIEndpoint of the Client. Relative URLs should always be specified without a preceding slash.
// If specified, the value pointed to by body is JSON encoded and included as the request body.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	if !strings.HasSuffix(c.APIEndpoint.Path, "/") {
		return nil, fmt.Errorf("APIEndpoint must have a trailing slash, but %q does not", c.APIEndpoint)
	}
	u, err := c.APIEndpoint.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if body != nil {
		err = json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Accept", mediaType)
	req.Header.Set("Content-Type", mediaType)
	req.Header.Set("User-Agent", c.UserAgent)

	return req, nil
}

// Do sends an API request and returns the API response. The API response is JSON decoded and stored in
// the value pointed to by v, or returned as an error if an API error has occurred.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	req = req.WithContext(ctx)
	resp, err := c.client.Do(req)
	if err != nil {
		// if we got an error, and the context has been canceled, the context's error is more useful.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	response := newResponse(resp)
	err = CheckResponse(response)
	if err != nil {
		return response, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, resp.Body)
			if err != nil {
				return nil, err
			}
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
			if err != nil {
				return nil, err
			}
		}
	}

	return response, err
}

// newResponse creates a new Response for the provided http.Response. r must be not nil.
func newResponse(r *http.Response) *Response {
	response := &Response{Response: r}
	response.populateRequestID()
	return response
}

// populateRequestID parses the request headers and populates the response request id.
func (r *Response) populateRequestID() {
	if requestID := r.Header.Get(headerRequestID); requestID != "" {
		r.RequestID = requestID
	}
}

// CheckResponse checks the API response for errors, and returns them if present. A response is considered
// an error if it has a status code outside the 200 range.
func CheckResponse(resp *Response) error {
	if code := resp.StatusCode; code >= 200 && code <= 299 {
		return nil
	}

	errorResponse := &ErrorResponse{Response: resp}

	data, err := ioutil.ReadAll(resp.Body)
	if err == nil && len(data) > 0 {
		err := json.Unmarshal(data, &errorResponse.Errors)
		if err != nil {
			return err
		}
	}
	return errorResponse
}
