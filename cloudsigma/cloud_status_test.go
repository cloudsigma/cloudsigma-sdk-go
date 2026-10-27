package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudStatus_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/cloud_status/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"guest":true,"trial":true}`)
	})
	expected := &CloudStatus{
		Guest: true,
		Trial: true,
	}

	cloudStatus, _, err := client.CloudStatus.Get(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, cloudStatus)
}

func TestCloudStatus_GetHTTPError(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/cloud_status/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = fmt.Fprint(w, `{"error":"synthetic"}`)
	})

	cloudStatus, resp, err := client.CloudStatus.Get(ctx)
	assert.Error(t, err)
	assert.Nil(t, cloudStatus)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestCloudStatus_GetContextCancellation(t *testing.T) {
	setup()
	defer teardown()

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	cloudStatus, resp, err := client.CloudStatus.Get(canceled)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, cloudStatus)
	assert.Nil(t, resp)
}
