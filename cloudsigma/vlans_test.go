package cloudsigma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVLANs_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/vlans/detail/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"uuid":"long-uuid"}],"meta":{"total_count":1}}`)
	})
	expected := []VLAN{
		{
			UUID: "long-uuid",
		},
	}

	vlans, resp, err := client.VLANs.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, vlans)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestVLANs_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/vlans/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"meta":{"key":"value"},"uuid":"long-uuid"}`)
	})
	expected := &VLAN{
		Meta: map[string]interface{}{
			"key": "value",
		},
		UUID: "long-uuid",
	}

	vlan, _, err := client.VLANs.Get(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, vlan)
}

func TestVLANs_Get_emptyUUID(t *testing.T) {
	_, _, err := client.VLANs.Get(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestVLANs_Get_invalidUUID(t *testing.T) {
	_, _, err := client.VLANs.Get(ctx, "%")

	assert.Error(t, err)
}

func TestVLANs_Update(t *testing.T) {
	setup()
	defer teardown()

	input := &VLANUpdateRequest{
		VLAN: &VLAN{
			Meta: map[string]interface{}{
				"meta-key": "meta-value",
				"locked":   "True",
			},
		},
	}
	mux.HandleFunc("/vlans/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		v := new(VLANUpdateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPut, r.Method)
		_, _ = fmt.Fprint(w, `{"meta":{"meta-key":"meta-value","locked":"True"},"uuid":"long-uuid"}`)
	})
	expected := &VLAN{
		Meta: map[string]interface{}{
			"meta-key": "meta-value",
			"locked":   "True",
		},
		UUID: "long-uuid",
	}

	vlan, _, err := client.VLANs.Update(ctx, "long-uuid", input)

	assert.NoError(t, err)
	assert.Equal(t, expected, vlan)
}

func TestVLANs_Update_emptyUUID(t *testing.T) {
	input := &VLANUpdateRequest{
		VLAN: &VLAN{
			Meta: map[string]interface{}{
				"meta-key": "meta-value",
			},
		},
	}

	_, _, err := client.VLANs.Update(ctx, "", input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestVLANs_Update_invalidUUID(t *testing.T) {
	input := &VLANUpdateRequest{
		VLAN: &VLAN{
			Meta: map[string]interface{}{
				"meta-key": "meta-value",
			},
		},
	}

	_, _, err := client.VLANs.Update(ctx, "%", input)

	assert.Error(t, err)
}

func TestVLANs_Update_emptyPayload(t *testing.T) {
	_, _, err := client.VLANs.Update(ctx, "long-uuid", nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}
