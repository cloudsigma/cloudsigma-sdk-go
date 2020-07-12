package cloudsigma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTags_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/tags/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test tag"}],"meta":{"total_count":1}}`)
	})
	expected := []Tag{
		{
			Name: "test tag",
		},
	}

	tags, resp, err := client.Tags.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, tags)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestTags_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/tags/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test tag","uuid":"long-uuid"}`)
	})
	expected := &Tag{
		Name: "test tag",
		UUID: "long-uuid",
	}

	tag, _, err := client.Tags.Get(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, tag)
}

func TestTags_Get_emptyUUID(t *testing.T) {
	_, _, err := client.Tags.Get(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestTags_Get_invalidUUID(t *testing.T) {
	_, _, err := client.Tags.Get(ctx, "%")

	assert.Error(t, err)
}

func TestTags_Create(t *testing.T) {
	setup()
	defer teardown()

	input := &TagCreateRequest{
		Tags: []Tag{
			{Name: "test tag"},
		},
	}
	mux.HandleFunc("/tags/", func(w http.ResponseWriter, r *http.Request) {
		v := new(TagCreateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test tag","uuid":"long-uuid"}]}`)

	})
	expected := []Tag{
		{Name: "test tag", UUID: "long-uuid"},
	}

	tags, _, err := client.Tags.Create(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, expected, tags)
}

func TestTags_Create_emptyPayload(t *testing.T) {
	_, _, err := client.Tags.Create(ctx, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestTags_Update(t *testing.T) {
	setup()
	defer teardown()

	input := &TagUpdateRequest{
		Tag: &Tag{
			Name: "test tag v2",
		},
	}
	mux.HandleFunc("/tags/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		v := new(TagUpdateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPut, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test tag v2","uuid":"long-uuid"}`)
	})
	expected := &Tag{
		Name: "test tag v2",
		UUID: "long-uuid",
	}

	tag, _, err := client.Tags.Update(ctx, "long-uuid", input)

	assert.NoError(t, err)
	assert.Equal(t, expected, tag)
}

func TestTags_Update_emptyUUID(t *testing.T) {
	input := &TagUpdateRequest{
		Tag: &Tag{
			Name: "test tag v2",
		},
	}

	_, _, err := client.Tags.Update(ctx, "", input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestTags_Update_invalidUUID(t *testing.T) {
	input := &TagUpdateRequest{
		Tag: &Tag{
			Name: "test tag v2",
		},
	}

	_, _, err := client.Tags.Update(ctx, "%", input)

	assert.Error(t, err)
}

func TestTags_Update_emptyPayload(t *testing.T) {
	_, _, err := client.Tags.Update(ctx, "long-uuid", nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestTags_Delete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/tags/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
	})

	_, err := client.Tags.Delete(ctx, "long-uuid")

	assert.NoError(t, err)
}

func TestTags_Delete_emptyUUID(t *testing.T) {
	_, err := client.Tags.Delete(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}
