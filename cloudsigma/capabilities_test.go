package cloudsigma

import (
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
		Hypervisors: CapabilitiesHypervisors{
			KVM: []string{"amd"},
		},
	}

	capabilities, _, err := client.Capabilities.Get(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, capabilities)
}
