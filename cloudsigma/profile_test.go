package cloudsigma

import (
	"context"
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

func TestProfile_GetHTTPError(t *testing.T) {
	setup()
	defer teardown()
	mux.HandleFunc("/profile/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error":"synthetic"}`)
	})
	profile, resp, err := client.Profile.Get(ctx)
	assert.Error(t, err)
	assert.Nil(t, profile)
	assert.NotNil(t, resp)
}

func TestProfile_GetContextCancellation(t *testing.T) {
	setup()
	defer teardown()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	profile, resp, err := client.Profile.Get(canceled)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, profile)
	assert.Nil(t, resp)
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

func TestProfile_UpdateHTTPError(t *testing.T) {
	setup()
	defer teardown()
	mux.HandleFunc("/profile/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error":"synthetic"}`)
	})
	profile, resp, err := client.Profile.Update(ctx, &ProfileUpdateRequest{Profile: &Profile{Company: "test"}})
	assert.Error(t, err)
	assert.Nil(t, profile)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestProfile_UpdateContextCancellation(t *testing.T) {
	setup()
	defer teardown()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	profile, resp, err := client.Profile.Update(canceled, &ProfileUpdateRequest{Profile: &Profile{Company: "test"}})
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, profile)
	assert.Nil(t, resp)
}
