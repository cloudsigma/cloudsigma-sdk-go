package cloudsigma

import (
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
