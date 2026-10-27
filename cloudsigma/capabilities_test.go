package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapabilities_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/capabilities/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"hypervisors":{"kvm":["amd"]}}`)
	})
	expected := &Capabilities{
		Hypervisors: &CapabilitiesHypervisors{
			KVM: []string{"amd"},
		},
	}

	capabilities, _, err := client.Capabilities.Get(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, capabilities)
}

func TestCapabilities_GetHTTPError(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/capabilities/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error":"synthetic"}`)
	})

	capabilities, resp, err := client.Capabilities.Get(ctx)
	assert.Error(t, err)
	assert.Nil(t, capabilities)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestCapabilities_GetContextCancellation(t *testing.T) {
	setup()
	defer teardown()

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	capabilities, resp, err := client.Capabilities.Get(canceled)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, capabilities)
	assert.Nil(t, resp)
}
