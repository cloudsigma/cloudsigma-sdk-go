package cloudsigma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirewallPolicies_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/fwpolicies/detail/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"uuid":"long-uuid"}],"meta":{"total_count":1}}`)
	})
	expected := []FirewallPolicy{
		{
			UUID: "long-uuid",
		},
	}

	policies, resp, err := client.FirewallPolicies.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, policies)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestFirewallPolicies_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/fwpolicies/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test fwpolicy","uuid":"long-uuid"}`)
	})
	expected := &FirewallPolicy{
		Name: "test fwpolicy",
		UUID: "long-uuid",
	}

	policy, _, err := client.FirewallPolicies.Get(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, policy)
}

func TestFirewallPolicies_Get_emptyUUID(t *testing.T) {
	_, _, err := client.FirewallPolicies.Get(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestFirewallPolicies_Get_invalidUUID(t *testing.T) {
	_, _, err := client.FirewallPolicies.Get(ctx, "%")

	assert.Error(t, err)
}

func TestFirewallPolicies_Create(t *testing.T) {
	setup()
	defer teardown()

	input := &FirewallPolicyCreateRequest{
		FirewallPolicies: []FirewallPolicy{
			{Name: "test policy"},
		},
	}
	mux.HandleFunc("/fwpolicies/", func(w http.ResponseWriter, r *http.Request) {
		v := new(FirewallPolicyCreateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test policy","uuid":"long-uuid"}]}`)
	})
	expected := []FirewallPolicy{
		{Name: "test policy", UUID: "long-uuid"},
	}

	policies, _, err := client.FirewallPolicies.Create(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, expected, policies)
}

func TestFirewallPolicies_Create_emptyPayload(t *testing.T) {
	_, _, err := client.FirewallPolicies.Create(ctx, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestFirewallPolicies_Update(t *testing.T) {
	setup()
	defer teardown()

	input := &FirewallPolicyUpdateRequest{
		FirewallPolicy: &FirewallPolicy{
			Name: "test policy v2",
		},
	}
	mux.HandleFunc("/fwpolicies/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		v := new(FirewallPolicyUpdateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPut, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test policy v2","uuid":"long-uuid"}`)
	})
	expected := &FirewallPolicy{
		Name: "test policy v2",
		UUID: "long-uuid",
	}

	policy, _, err := client.FirewallPolicies.Update(ctx, "long-uuid", input)

	assert.NoError(t, err)
	assert.Equal(t, expected, policy)
}

func TestFirewallPolicies_Update_emptyUUID(t *testing.T) {
	input := &FirewallPolicyUpdateRequest{
		FirewallPolicy: &FirewallPolicy{
			UUID: "long-uuid",
		},
	}

	_, _, err := client.FirewallPolicies.Update(ctx, "", input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestFirewallPolicies_Update_invalidUUID(t *testing.T) {
	input := &FirewallPolicyUpdateRequest{
		FirewallPolicy: &FirewallPolicy{
			UUID: "long-uuid",
		},
	}

	_, _, err := client.FirewallPolicies.Update(ctx, "%", input)

	assert.Error(t, err)
}

func TestFirewallPolicies_Update_emptyPayload(t *testing.T) {
	_, _, err := client.FirewallPolicies.Update(ctx, "long-uuid", nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestFirewallPolicies_Delete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/fwpolicies/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
	})

	_, err := client.FirewallPolicies.Delete(ctx, "long-uuid")

	assert.NoError(t, err)
}

func TestFirewallPolicies_Delete_emptyUUID(t *testing.T) {
	_, err := client.FirewallPolicies.Delete(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}
