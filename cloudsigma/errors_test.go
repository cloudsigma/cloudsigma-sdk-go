package cloudsigma

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors_Error_messageFormat(t *testing.T) {
	errorResponse := &ErrorResponse{
		Response: &Response{
			Response: &http.Response{
				Request: &http.Request{
					Method: http.MethodGet,
					URL:    &url.URL{Scheme: "https", Path: "cloudsigma.com/api"},
				},
				StatusCode: 200,
			},
		},
		Errors: []Error{
			{Message: "first", Type: "permission"},
			{Message: "second"},
		},
	}
	expectedMessage := "GET https://cloudsigma.com/api: 200 [{Message:first Point: Type:permission} {Message:second Point: Type:}]"

	assert.Error(t, errorResponse)
	assert.Equal(t, expectedMessage, errorResponse.Error())
}

func TestErrors_Error_quotedRequestID(t *testing.T) {
	errorResponse := &ErrorResponse{
		Response: &Response{
			Response: &http.Response{
				Request: &http.Request{
					Method: http.MethodGet,
					URL:    &url.URL{Scheme: "https", Path: "cloudsigma.com/api"},
				},
				StatusCode: 500,
			},
		},
		Errors: []Error{
			{Message: "unknown error", Type: "backend"},
		},
	}
	errorResponse.Response.RequestID = "long-long-uuid"
	expectedMessage := "GET https://cloudsigma.com/api: 500 (request \"long-long-uuid\") [{Message:unknown error Point: Type:backend}]"

	assert.Error(t, errorResponse)
	assert.Equal(t, expectedMessage, errorResponse.Error())
}
