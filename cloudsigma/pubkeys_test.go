package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPubkeys_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/pubkeys/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"has_private_key":true,"name":"test key"}],"meta":{"total_count":1}}`)
	})
	expected := []Keypair{
		{
			HasPrivateKey: true,
			Name:          "test key",
		},
	}

	keypairs, resp, err := client.Pubkeys.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, keypairs)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestPubkeys_ListHTTPError(t *testing.T) {
	setup()
	defer teardown()
	mux.HandleFunc("/pubkeys/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/pubkeys/", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `[{"error_type":"synthetic","error_message":"denied","error_point":"list"}]`)
	})

	keypairs, resp, err := client.Pubkeys.List(ctx)
	var apiErr *ErrorResponse
	assert.ErrorAs(t, err, &apiErr)
	assert.Nil(t, keypairs)
	if assert.NotNil(t, resp) {
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	}
	if assert.NotNil(t, apiErr) {
		assert.Same(t, resp, apiErr.Response)
		assert.Len(t, apiErr.Errors, 1)
		assert.Equal(t, "synthetic", apiErr.Errors[0].Type)
		assert.Equal(t, "denied", apiErr.Errors[0].Message)
		assert.Equal(t, "list", apiErr.Errors[0].Point)
	}
}

func TestPubkeys_ListContextCancellation(t *testing.T) {
	setup()
	defer teardown()
	canceled, cancel := context.WithCancel(ctx)
	cancel()

	keypairs, resp, err := client.Pubkeys.List(canceled)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, keypairs)
	assert.Nil(t, resp)
}

func TestPubkeys_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/pubkeys/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test key","uuid":"long-uuid"}`)
	})
	expected := &Keypair{
		Name: "test key",
		UUID: "long-uuid",
	}

	keypair, _, err := client.Pubkeys.Get(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, keypair)
}

func TestPubkeys_Get_emptyUUID(t *testing.T) {
	_, _, err := client.Pubkeys.Get(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestPubkeys_Get_invalidUUID(t *testing.T) {
	_, _, err := client.Pubkeys.Get(ctx, "%")

	assert.Error(t, err)
}
