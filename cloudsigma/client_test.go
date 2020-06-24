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

	client = NewBasicAuthClient("user", "password")
	client.BaseURL, _ = url.Parse(server.URL + "/")
}

func teardown() {
	server.Close()
}

func TestClient_NewBasicAuthClient(t *testing.T) {
	setup()
	defer teardown()

	assert.NotNil(t, client.BaseURL)
	assert.Equal(t, server.URL+"/", client.BaseURL.String())
	assert.Equal(t, "user", client.Username)
	assert.Equal(t, "password", client.Password)
}

func TestClient_SetLocationForBaseURL_customLocation(t *testing.T) {
	setup()
	defer teardown()

	client.SetLocationForBaseURL("wdc")

	assert.Equal(t, "https://wdc.cloudsigma.com/api/2.0/", client.BaseURL.String())
}

func TestClient_SetLocationForBaseURL_emptyLocation(t *testing.T) {
	setup()
	defer teardown()

	client.SetLocationForBaseURL("")

	assert.Equal(t, "https://zrh.cloudsigma.com/api/2.0/", client.BaseURL.String())
}

func TestClient_NewRequest(t *testing.T) {
	setup()
	defer teardown()

	client.BaseURL, _ = url.Parse("https://zrh.cloudsigma.com/api/2.0/")
	req, err := client.NewRequest("GET", "ips/uuid", nil)

	assert.NoError(t, err)
	assert.Equal(t, "https://zrh.cloudsigma.com/api/2.0/ips/uuid", req.URL.String())
}

func TestClient_NewRequest_baseURLWithoutTrailingSlash(t *testing.T) {
	setup()
	defer teardown()

	client.BaseURL, _ = url.Parse("https://zrh.cloudsigma.com/api/2.0")
	_, err := client.NewRequest("GET", "ips/uuid", nil)

	assert.Error(t, err)
}

func TestClient_NewRequest_invalidRequestURL(t *testing.T) {
	setup()
	defer teardown()

	client.BaseURL, _ = url.Parse("/")
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

func TestClient_CheckResponse_errorElements(t *testing.T) {
	resp := &http.Response{
		Request:    &http.Request{},
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(strings.NewReader(`[{"error_message":"error"}]`)),
	}
	expected := []ErrorElement{
		{Message: "error"},
	}

	err := CheckResponse(resp).(*ErrorResponse)

	assert.Error(t, err)
	assert.Equal(t, 400, err.Response.StatusCode)
	assert.Equal(t, expected, err.ErrorElements)
}

func TestClient_CheckResponse_errorWhenUnmarshall(t *testing.T) {
	resp := &http.Response{
		Request:    &http.Request{},
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(strings.NewReader(`{"error_message":"response is always an array of errors"}`)),
	}

	err := CheckResponse(resp).(*json.UnmarshalTypeError)

	assert.Error(t, err)
}

func TestClient_CheckResponse_noBody(t *testing.T) {
	resp := &http.Response{
		Request:    &http.Request{},
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(strings.NewReader("")),
	}

	err := CheckResponse(resp).(*ErrorResponse)

	assert.Error(t, err)
	assert.Equal(t, 400, err.Response.StatusCode)
	assert.Nil(t, err.ErrorElements)
}

func TestClient_CheckResponse_noErrorStatusCode(t *testing.T) {
	resp := &http.Response{
		Request:    &http.Request{},
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader("")),
	}

	err := CheckResponse(resp)

	assert.NoError(t, err)
}
