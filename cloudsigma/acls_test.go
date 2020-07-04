package cloudsigma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestACLs_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/acls/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"uuid":"long-uuid"}],"meta":{"total_count":1}}`)
	})
	expected := []ACL{
		{
			UUID: "long-uuid",
		},
	}

	acls, resp, err := client.ACLs.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, acls)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestACLs_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/acls/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"meta":{"key":"value"},"uuid":"long-uuid"}`)
	})
	expected := &ACL{
		Meta: map[string]interface{}{
			"key": "value",
		},
		UUID: "long-uuid",
	}

	acl, _, err := client.ACLs.Get(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, acl)
}

func TestACLs_Get_emptyUUID(t *testing.T) {
	_, _, err := client.ACLs.Get(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestACLs_Get_invalidUUID(t *testing.T) {
	_, _, err := client.ACLs.Get(ctx, "%")

	assert.Error(t, err)
}

func TestACLs_Create(t *testing.T) {
	setup()
	defer teardown()

	input := &ACLCreateRequest{
		ACLs: []ACL{
			{Name: "test acl"},
		},
	}
	mux.HandleFunc("/acls/", func(w http.ResponseWriter, r *http.Request) {
		v := new(ACLCreateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test acl","uuid":"long-uuid"}]}`)
	})
	expected := []ACL{
		{Name: "test acl", UUID: "long-uuid"},
	}

	acls, _, err := client.ACLs.Create(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, expected, acls)
}

func TestACLs_Create_emptyPayload(t *testing.T) {
	_, _, err := client.ACLs.Create(ctx, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestACLs_Update(t *testing.T) {
	setup()
	defer teardown()

	input := &ACLUpdateRequest{
		ACL: &ACL{
			Meta: map[string]interface{}{
				"meta-key": "meta-value",
				"locked":   "True",
			},
		},
	}
	mux.HandleFunc("/acls/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		v := new(ACLUpdateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPut, r.Method)
		_, _ = fmt.Fprint(w, `{"meta":{"meta-key":"meta-value","locked":"True"},"uuid":"long-uuid"}`)
	})
	expected := &ACL{
		Meta: map[string]interface{}{
			"meta-key": "meta-value",
			"locked":   "True",
		},
		UUID: "long-uuid",
	}

	acl, _, err := client.ACLs.Update(ctx, "long-uuid", input)

	assert.NoError(t, err)
	assert.Equal(t, expected, acl)
}

func TestACLs_Update_emptyUUID(t *testing.T) {
	input := &ACLUpdateRequest{
		ACL: &ACL{
			UUID: "long-uuid",
		},
	}

	_, _, err := client.ACLs.Update(ctx, "", input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestACLs_Update_invalidUUID(t *testing.T) {
	input := &ACLUpdateRequest{
		ACL: &ACL{
			UUID: "long-uuid",
		},
	}

	_, _, err := client.ACLs.Update(ctx, "%", input)

	assert.Error(t, err)
}

func TestACLs_Update_emptyPayload(t *testing.T) {
	_, _, err := client.ACLs.Update(ctx, "long-uuid", nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestACLs_Delete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/acls/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
	})

	_, err := client.ACLs.Delete(ctx, "long-uuid")

	assert.NoError(t, err)
}

func TestACLs_Delete_emptyUUID(t *testing.T) {
	_, err := client.ACLs.Delete(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}
