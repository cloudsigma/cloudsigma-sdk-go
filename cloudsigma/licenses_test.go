package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLicenses_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/licenses/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test license"}],"meta":{"total_count":1}}`)
	})
	expected := []License{
		{
			Name: "test license",
		},
	}

	licenses, resp, err := client.Licenses.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, licenses)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestLicenses_ListHTTPError(t *testing.T) {
	setup()
	defer teardown()
	mux.HandleFunc("/licenses/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error":"synthetic"}`)
	})
	licenses, resp, err := client.Licenses.List(ctx)
	assert.Error(t, err)
	assert.Nil(t, licenses)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestLicenses_ListContextCancellation(t *testing.T) {
	setup()
	defer teardown()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	licenses, resp, err := client.Licenses.List(canceled)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, licenses)
	assert.Nil(t, resp)
}
