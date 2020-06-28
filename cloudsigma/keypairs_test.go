package cloudsigma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeypairs_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/keypairs/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test key","public_key":"long-public-key"}],"meta":{"total_count":1}}`)
	})
	expected := []Keypair{
		{
			Name:      "test key",
			PublicKey: "long-public-key",
		},
	}

	keypairs, resp, err := client.Keypairs.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, keypairs)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestKeypairs_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/keypairs/long-uuid", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test key","uuid":"long-uuid"}`)
	})
	expected := &Keypair{
		Name: "test key",
		UUID: "long-uuid",
	}

	keypair, _, err := client.Keypairs.Get(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, keypair)
}

func TestKeypairs_Get_emptyUUID(t *testing.T) {
	_, _, err := client.Keypairs.Get(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestKeypairs_Get_invalidUUID(t *testing.T) {
	_, _, err := client.Keypairs.Get(ctx, "%")

	assert.Error(t, err)
}

func TestKeypairs_Create(t *testing.T) {
	setup()
	defer teardown()

	input := &KeypairCreateRequest{
		Keypairs: []Keypair{
			{Name: "test keypair"},
		},
	}
	mux.HandleFunc("/keypairs/", func(w http.ResponseWriter, r *http.Request) {
		v := new(KeypairCreateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test keypair","public_key":"long-long-public-key"}]}`)
	})
	expected := []Keypair{
		{Name: "test keypair", PublicKey: "long-long-public-key"},
	}

	keypairs, _, err := client.Keypairs.Create(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, expected, keypairs)
}

func TestKeypairs_Create_emptyPayload(t *testing.T) {
	_, _, err := client.Keypairs.Create(ctx, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestKeypairs_Update(t *testing.T) {
	setup()
	defer teardown()

	input := &KeypairUpdateRequest{
		Keypair: &Keypair{
			Name: "test keypair v2",
		},
	}
	mux.HandleFunc("/keypairs/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		v := new(KeypairUpdateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPut, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test keypair v2","uuid":"long-uuid"}`)
	})
	expected := &Keypair{
		Name: "test keypair v2",
		UUID: "long-uuid",
	}

	keypair, _, err := client.Keypairs.Update(ctx, "long-uuid", input)

	assert.NoError(t, err)
	assert.Equal(t, expected, keypair)
}

func TestKeypairs_Update_emptyUUID(t *testing.T) {
	input := &KeypairUpdateRequest{
		Keypair: &Keypair{
			Name: "test keypair v2",
		},
	}

	_, _, err := client.Keypairs.Update(ctx, "", input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestKeypairs_Update_invalidUUID(t *testing.T) {
	input := &KeypairUpdateRequest{
		Keypair: &Keypair{
			Name: "test keypair v2",
		},
	}

	_, _, err := client.Keypairs.Update(ctx, "%", input)

	assert.Error(t, err)
}

func TestKeypairs_Update_emptyPayload(t *testing.T) {
	_, _, err := client.Keypairs.Update(ctx, "long-uuid", nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestKeypairs_Delete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/keypairs/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
	})

	_, err := client.Keypairs.Delete(ctx, "long-uuid")

	assert.NoError(t, err)
}

func TestKeypairs_Delete_emptyUUID(t *testing.T) {
	_, err := client.Keypairs.Delete(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}
