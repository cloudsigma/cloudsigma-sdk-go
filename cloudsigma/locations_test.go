package cloudsigma

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocations_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/locations/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"id":"ZRH"}],"meta":{"total_count":1}}`)
	})
	expected := []Location{
		{
			ID: "ZRH",
		},
	}

	locations, resp, err := client.Locations.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, locations)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}
