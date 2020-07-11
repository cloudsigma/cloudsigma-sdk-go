package cloudsigma

import (
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
