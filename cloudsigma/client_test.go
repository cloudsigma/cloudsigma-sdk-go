package cloudsigma

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	mux    *http.ServeMux
	ctx    = context.TODO()
	client *Client
	server *httptest.Server
)

func setup() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	cred := NewUsernamePasswordCredentialsProvider("user", "password")
	var opts []ClientOption
	client = NewClient(cred, opts...)
	client.baseURL, _ = url.Parse(fmt.Sprintf("%v/", server.URL))
}

func setupWithToken() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	cred := NewTokenCredentialsProvider("access_token")
	var opts []ClientOption
	client = NewClient(cred, opts...)
	client.baseURL, _ = url.Parse(fmt.Sprintf("%v/", server.URL))
}

func teardown() {
	server.Close()
}

func TestClient_addOptions(t *testing.T) {
	path := "/servers/"
	opts := &ListOptions{Limit: 25, Offset: 5}

	pathWithOpts, err := addOptions(path, opts)

	assert.NoError(t, err)
	assert.Equal(t, "/servers/?limit=25&offset=5", pathWithOpts)
}

func TestClient_Defaults(t *testing.T) {
	client := NewClient(nil)

	assert.Equal(t, "https://zrh.cloudsigma.com/api/2.0/", client.baseURL.String())
	assert.Contains(t, client.userAgent, "cloudsigma-sdk-go/")
	assert.Equal(t, 0*time.Second, client.httpClient.Timeout)
}

func TestClient_WithHTTPClient(t *testing.T) {
	httpClient := &http.Client{Timeout: 2 * time.Second}
	client := NewClient(nil, WithHTTPClient(httpClient))

	assert.Equal(t, httpClient, client.httpClient)
	assert.Equal(t, 2*time.Second, client.httpClient.Timeout)
}

func TestClient_WithLocation(t *testing.T) {
	expectedBaseURL, _ := url.Parse("https://wdc.cloudsigma.com/api/2.0/")
	client := NewClient(nil, WithLocation("wdc"))

	assert.Equal(t, expectedBaseURL, client.baseURL)
}

func TestClient_WithUserAgent(t *testing.T) {
	expectedUserAgent := "terraform-provider-cloudsigma/1.1.0-release"
	client := NewClient(nil, WithUserAgent("terraform-provider-cloudsigma/1.1.0-release"))

	assert.Equal(t, expectedUserAgent, client.userAgent)
}

func TestClient_NewRequest(t *testing.T) {
	setup()
	defer teardown()

	req, err := client.NewRequest("GET", "ips/uuid", nil)

	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%v/ips/uuid", server.URL), req.URL.String())
}

func TestClient_NewRequest_withAccessToken(t *testing.T) {
	setupWithToken()
	defer teardown()

	req, err := client.NewRequest("GET", "ips/uuid", nil)

	assert.NoError(t, err)
	assert.Equal(t, "Bearer access_token", req.Header.Get("Authorization"))
	assert.Equal(t, fmt.Sprintf("%v/ips/uuid", server.URL), req.URL.String())
}

func TestClient_NewRequest_withUsernamePassword(t *testing.T) {
	setup()
	defer teardown()
	expectedAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:password"))

	req, err := client.NewRequest(http.MethodGet, "ips/uuid", nil)

	assert.Nil(t, err)
	assert.Equal(t, expectedAuthHeader, req.Header.Get("Authorization"))
	assert.Equal(t, fmt.Sprintf("%v/ips/uuid", server.URL), req.URL.String())
}

func TestClient_NewRequest_baseURLWithoutTrailingSlash(t *testing.T) {
	setup()
	defer teardown()

	client.baseURL, _ = url.Parse("https://zrh.cloudsigma.com/api/2.0")
	_, err := client.NewRequest("GET", "ips/uuid", nil)

	assert.Error(t, err)
}

func TestClient_NewRequest_invalidRequestURL(t *testing.T) {
	setup()
	defer teardown()

	client.baseURL, _ = url.Parse("/")
	_, err := client.NewRequest("GET", ":%31", nil)

	assert.Error(t, err)
}

func TestClient_Do(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		_, _ = fmt.Fprint(w, `{"A":"a"}`)
	})
	req, _ := client.NewRequest("GET", ".", nil)
	type foo struct {
		A string
	}
	body := new(foo)

	_, err := client.Do(ctx, req, body)
	expected := &foo{"a"}

	assert.NoError(t, err)
	assert.Equal(t, body, expected)
}

func TestClient_Do_httpError(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	})
	req, _ := client.NewRequest("GET", ".", nil)

	resp, err := client.Do(ctx, req, nil)

	assert.Error(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestClient_populateRequestID(t *testing.T) {
	resp := &Response{
		Response: &http.Response{
			Header: map[string][]string{},
		}}
	resp.Header.Set("X-REQUEST-ID", "long-uuid")

	resp.populateRequestID()

	assert.Equal(t, "long-uuid", resp.RequestID)
}

func TestClient_CheckResponse_errorElements(t *testing.T) {
	resp := &Response{
		Response: &http.Response{
			Request:    &http.Request{},
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(`[{"error_message":"error"}]`)),
		}}
	expected := []Error{
		{Message: "error"},
	}

	err := CheckResponse(resp).(*ErrorResponse)

	assert.Error(t, err)
	assert.Equal(t, 400, err.Response.StatusCode)
	assert.Equal(t, expected, err.Errors)
}

func TestClient_CheckResponse_errorWhenUnmarshall(t *testing.T) {
	resp := &Response{
		Response: &http.Response{
			Request:    &http.Request{},
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(`{"error_message":"response is always an array of errors"}`)),
		},
	}

	err := CheckResponse(resp).(*json.UnmarshalTypeError)

	assert.Error(t, err)
}

func TestClient_CheckResponse_noBody(t *testing.T) {
	resp := &Response{
		Response: &http.Response{
			Request:    &http.Request{},
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("")),
		},
	}

	err := CheckResponse(resp).(*ErrorResponse)

	assert.Error(t, err)
	assert.Equal(t, 400, err.Response.StatusCode)
	assert.Nil(t, err.Errors)
}

func TestClient_CheckResponse_noErrorStatusCode(t *testing.T) {
	resp := &Response{
		Response: &http.Response{
			Request:    &http.Request{},
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		},
	}

	err := CheckResponse(resp)

	assert.NoError(t, err)
}
