package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/google/go-querystring/query"
)

const (
	libraryVersion   = "0.14.0"
	defaultLocation  = "zrh"
	defaultUserAgent = "cloudsigma-sdk-go/" + libraryVersion

	// endpointURL is a URL with the placeholder for API location.
	endpointURL     = "https://%s.cloudsigma.com/api/2.0/"
	headerRequestID = "X-REQUEST-ID"
	mediaType       = "application/json"
)

// A Client manages communication with the CloudSigma API.
type Client struct {
	// Base URL for API requests of the CloudSigma API, e.g.:
	//   https://{location}.cloudsigma.com/api/2.0/
	// Where {location} is a subdomain for a specific location. All available locations
	// can be queried from Locations endpoint.
	baseURL *url.URL

	httpClient   *http.Client // HTTP client used to communicate with the API.
	credProvider CredentialsProvider
	userAgent    string // User agent used when communicating with the CloudSigma API.

	common service // Reuse a single struct instead of allocating one for each service on the heap.

	ACLs             *ACLsService
	Capabilities     *CapabilitiesService
	CloudStatus      *CloudStatusService
	Drives           *DrivesService
	FirewallPolicies *FirewallPoliciesService
	IPs              *IPsService
	Keypairs         *KeypairsService
	Licenses         *LicensesService
	LibraryDrives    *LibraryDrivesService
	Locations        *LocationsService
	Profile          *ProfileService
	Pubkeys          *PubkeysService
	RemoteSnapshots  *RemoteSnapshotsService
	Servers          *ServersService
	Snapshots        *SnapshotsService
	Subscriptions    *SubscriptionsService
	Tags             *TagsService
	VLANs            *VLANsService
}

type service struct {
	client *Client
}

// ListOptions specifies the optional parameters to various List methods that
// support offset pagination.
type ListOptions struct {
	// Limit specifies the maximum number of objects to be returned. If set to 0,
	// all resources will be returned. Note, there is no omitempty struct tag!
	Limit int `url:"limit"`

	// Offset specifies the index at which to start returning objects. It is
	// a zero based index.
	Offset int `url:"offset,omitempty"`
}

// addOptions adds the parameters in opts as URL query parameters to s. opts
// must be a struct whose fields may contain "url" tags.
func addOptions(s string, opts interface{}) (string, error) {
	v := reflect.ValueOf(opts)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opts)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

type ClientOption func(*Client)

// WithHTTPClient configures Client to use a specific http client for communication.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = httpClient
	}
}

// WithLocation configures Client to use a specific location.
func WithLocation(location string) ClientOption {
	return func(client *Client) {
		parsedURL, _ := url.Parse(fmt.Sprintf(endpointURL, location))
		client.baseURL = parsedURL
	}
}

// WithUserAgent configures Client to use a specific user agent.
func WithUserAgent(userAgent string) ClientOption {
	return func(client *Client) {
		client.userAgent = userAgent
	}
}

// NewClient returns a new CloudSigma API client.
func NewClient(cred CredentialsProvider, opts ...ClientOption) *Client {
	baseURL, _ := url.Parse(fmt.Sprintf(endpointURL, defaultLocation))
	httpClient := http.DefaultClient

	c := &Client{
		baseURL:      baseURL,
		httpClient:   httpClient,
		credProvider: cred,
		userAgent:    defaultUserAgent,
	}
	for _, opt := range opts {
		opt(c)
	}

	c.common.client = c

	c.ACLs = (*ACLsService)(&c.common)
	c.Capabilities = (*CapabilitiesService)(&c.common)
	c.CloudStatus = (*CloudStatusService)(&c.common)
	c.Drives = (*DrivesService)(&c.common)
	c.FirewallPolicies = (*FirewallPoliciesService)(&c.common)
	c.IPs = (*IPsService)(&c.common)
	c.Keypairs = (*KeypairsService)(&c.common)
	c.Licenses = (*LicensesService)(&c.common)
	c.LibraryDrives = (*LibraryDrivesService)(&c.common)
	c.Locations = (*LocationsService)(&c.common)
	c.Profile = (*ProfileService)(&c.common)
	c.Pubkeys = (*PubkeysService)(&c.common)
	c.RemoteSnapshots = (*RemoteSnapshotsService)(&c.common)
	c.Servers = (*ServersService)(&c.common)
	c.Snapshots = (*SnapshotsService)(&c.common)
	c.Subscriptions = (*SubscriptionsService)(&c.common)
	c.Tags = (*TagsService)(&c.common)
	c.VLANs = (*VLANsService)(&c.common)

	return c
}

// NewRequest creates an API request. A relative URL can be provided in urlStr, in which case it is resolved
// relative to the APIEndpoint of the Client. Relative URLs should always be specified without a preceding slash.
// If specified, the value pointed to by body is JSON encoded and included as the request body.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	if !strings.HasSuffix(c.baseURL.Path, "/") {
		return nil, fmt.Errorf("baseURL must have a trailing slash, but %q does not", c.baseURL)
	}
	u, err := c.baseURL.Parse(urlStr)
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

	credentials, err := c.credProvider.Retrieve()
	if err != nil {
		return nil, err
	}

	switch credentials.Source {
	case UsernamePasswordCredentialsName:
		req.SetBasicAuth(credentials.Username, credentials.Password)

	case TokenCredentialsName:
		req.Header.Set("Authorization", "Bearer "+credentials.Token)
	}

	req.Header.Set("Accept", mediaType)
	req.Header.Set("Content-Type", mediaType)
	req.Header.Set("User-Agent", c.userAgent)

	return req, nil
}

// Response is a CloudSigma response. This wraps the standard http.Response.
type Response struct {
	*http.Response

	Meta *Meta // Meta describes generic information about the response.

	RequestID string // RequestID returned from the API, useful to contact support.
}

// Do sends an API request and returns the API response. The API response is JSON decoded and stored in
// the value pointed to by v, or returned as an error if an API error has occurred.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	req = req.WithContext(ctx)
	resp, err := c.httpClient.Do(req)
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

	data, err := io.ReadAll(resp.Body)
	if err == nil && len(data) > 0 {
		err := json.Unmarshal(data, &errorResponse.Errors)
		if err != nil {
			return err
		}
	}
	return errorResponse
}
