package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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

	client = NewBasicAuthClient("user", "password", nil)
	client.APIEndpoint, _ = url.Parse(fmt.Sprintf("%v/", server.URL))
}

func setupWithToken() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	client = NewTokenClient("access_token", nil)
	client.APIEndpoint, _ = url.Parse(fmt.Sprintf("%v/", server.URL))
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

func TestClient_NewBasicAuthClient(t *testing.T) {
	setup()
	defer teardown()

	assert.NotNil(t, client.APIEndpoint)
	assert.Equal(t, fmt.Sprintf("%v/", server.URL), client.APIEndpoint.String())
	assert.Equal(t, "", client.Token)
	assert.Equal(t, "user", client.Username)
	assert.Equal(t, "password", client.Password)
	assert.Equal(t, defaultUserAgent, client.UserAgent)
}

func TestClient_NewTokenClient(t *testing.T) {
	setupWithToken()
	defer teardown()

	assert.NotNil(t, client.APIEndpoint)
	assert.Equal(t, fmt.Sprintf("%v/", server.URL), client.APIEndpoint.String())
	assert.Equal(t, "access_token", client.Token)
	assert.Equal(t, "", client.Username)
	assert.Equal(t, "", client.Password)
	assert.Equal(t, defaultUserAgent, client.UserAgent)
}

func TestClient_SetLocation(t *testing.T) {
	setup()
	defer teardown()

	client.SetLocation("wdc")

	assert.Equal(t, "https://wdc.cloudsigma.com/api/2.0/", client.APIEndpoint.String())
}

func TestClient_SetAPIEndpoint(t *testing.T) {
	setup()
	defer teardown()

	client.SetAPIEndpoint("some.custom.location", "custom-base-url.com/api/2.0/")

	assert.Equal(t, "https://some.custom.location.custom-base-url.com/api/2.0/", client.APIEndpoint.String())
}

func TestClient_SetAPIEndpoint_defaultLocation(t *testing.T) {
	setup()
	defer teardown()

	client.SetAPIEndpoint("", "custom-base-url.com/api/2.0/")

	assert.Equal(t, "https://zrh.custom-base-url.com/api/2.0/", client.APIEndpoint.String())
}

func TestClient_SetAPIEndpoint_defaultBaseURL(t *testing.T) {
	setup()
	defer teardown()

	client.SetAPIEndpoint("some.custom.location", "")

	assert.Equal(t, "https://some.custom.location.cloudsigma.com/api/2.0/", client.APIEndpoint.String())
}

func TestClient_SetUserAgent(t *testing.T) {
	setup()
	defer teardown()

	client.SetUserAgent("terraform-provider-cloudsigma/1.1.0-release")

	assert.Equal(t, "terraform-provider-cloudsigma/1.1.0-release", client.UserAgent)
}

func TestClient_NewRequest(t *testing.T) {
	setup()
	defer teardown()

	req, err := client.NewRequest("GET", "ips/uuid", nil)

	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%v/ips/uuid", server.URL), req.URL.String())
}

func TestClient_NewRequestWithToken(t *testing.T) {
	setupWithToken()
	defer teardown()

	req, err := client.NewRequest("GET", "ips/uuid", nil)

	assert.NoError(t, err)
	assert.Equal(t, "Bearer access_token", req.Header.Get("Authorization"))
	assert.Equal(t, fmt.Sprintf("%v/ips/uuid", server.URL), req.URL.String())
}

func TestClient_NewRequest_baseURLWithoutTrailingSlash(t *testing.T) {
	setup()
	defer teardown()

	client.APIEndpoint, _ = url.Parse("https://zrh.cloudsigma.com/api/2.0")
	_, err := client.NewRequest("GET", "ips/uuid", nil)

	assert.Error(t, err)
}

func TestClient_NewRequest_invalidRequestURL(t *testing.T) {
	setup()
	defer teardown()

	client.APIEndpoint, _ = url.Parse("/")
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
			Body:       ioutil.NopCloser(strings.NewReader(`[{"error_message":"error"}]`)),
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
			Body:       ioutil.NopCloser(strings.NewReader(`{"error_message":"response is always an array of errors"}`)),
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
			Body:       ioutil.NopCloser(strings.NewReader("")),
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
			Body:       ioutil.NopCloser(strings.NewReader("")),
		},
	}

	err := CheckResponse(resp)

	assert.NoError(t, err)
}
