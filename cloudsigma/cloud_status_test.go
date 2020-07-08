package cloudsigma

import (
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
