package cloudsigma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfile_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/profile/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"first_name":"John","last_name":"Doe"}`)
	})
	expected := &Profile{
		FirstName: "John",
		LastName:  "Doe",
	}

	profile, _, err := client.Profile.Get(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, profile)
}

func TestProfile_Update(t *testing.T) {
	setup()
	defer teardown()

	input := &ProfileUpdateRequest{
		Profile: &Profile{
			Company: "CloudSigma AG",
		},
	}
	mux.HandleFunc("/profile/", func(w http.ResponseWriter, r *http.Request) {
		v := new(ProfileUpdateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPut, r.Method)
		_, _ = fmt.Fprint(w, `{"company":"CloudSigma AG"}`)
	})
	expected := &Profile{
		Company: "CloudSigma AG",
	}

	profile, _, err := client.Profile.Update(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, expected, profile)
}

func TestProfile_Update_emptyPayload(t *testing.T) {
	_, _, err := client.Profile.Update(ctx, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}
