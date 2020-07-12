package cloudsigma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshots_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/snapshots/detail/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"uuid":"long-uuid"}],"meta":{"total_count":1}}`)
	})
	expected := []Snapshot{
		{
			UUID: "long-uuid",
		},
	}

	snapshots, resp, err := client.Snapshots.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, snapshots)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestSnapshots_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/snapshots/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"meta":{"key":"value"},"uuid":"long-uuid"}`)
	})
	expected := &Snapshot{
		Meta: map[string]interface{}{
			"key": "value",
		},
		UUID: "long-uuid",
	}

	snapshot, _, err := client.Snapshots.Get(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, snapshot)
}

func TestSnapshots_Get_emptyUUID(t *testing.T) {
	_, _, err := client.Snapshots.Get(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestSnapshots_Get_invalidUUID(t *testing.T) {
	_, _, err := client.Snapshots.Get(ctx, "%")

	assert.Error(t, err)
}

func TestSnapshots_Create(t *testing.T) {
	setup()
	defer teardown()

	input := &SnapshotCreateRequest{
		Snapshots: []Snapshot{
			{Name: "test snapshot"},
		},
	}
	mux.HandleFunc("/snapshots/", func(w http.ResponseWriter, r *http.Request) {
		v := new(SnapshotCreateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test snapshot","uuid":"long-uuid"}]}`)
	})
	expected := []Snapshot{
		{Name: "test snapshot", UUID: "long-uuid"},
	}

	snapshots, _, err := client.Snapshots.Create(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, expected, snapshots)
}

func TestSnapshots_Create_emptyPayload(t *testing.T) {
	_, _, err := client.Snapshots.Create(ctx, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestSnapshots_Update(t *testing.T) {
	setup()
	defer teardown()

	input := &SnapshotUpdateRequest{
		Snapshot: &Snapshot{
			Name: "test snapshot v2",
		},
	}
	mux.HandleFunc("/snapshots/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		v := new(SnapshotUpdateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPut, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test snapshot v2","uuid":"long-uuid"}`)
	})
	expected := &Snapshot{
		Name: "test snapshot v2",
		UUID: "long-uuid",
	}

	snapshot, _, err := client.Snapshots.Update(ctx, "long-uuid", input)

	assert.NoError(t, err)
	assert.Equal(t, expected, snapshot)
}

func TestSnapshots_Update_emptyUUID(t *testing.T) {
	input := &SnapshotUpdateRequest{
		Snapshot: &Snapshot{
			Name: "test snapshot v2",
		},
	}

	_, _, err := client.Snapshots.Update(ctx, "", input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestSnapshots_Update_invalidUUID(t *testing.T) {
	input := &SnapshotUpdateRequest{
		Snapshot: &Snapshot{
			Name: "test snapshot v2",
		},
	}

	_, _, err := client.Snapshots.Update(ctx, "%", input)

	assert.Error(t, err)
}

func TestSnapshots_Update_emptyPayload(t *testing.T) {
	_, _, err := client.Snapshots.Update(ctx, "long-uuid", nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestSnapshots_Delete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/snapshots/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
	})

	_, err := client.Snapshots.Delete(ctx, "long-uuid")

	assert.NoError(t, err)
}

func TestSnapshots_Delete_emptyUUID(t *testing.T) {
	_, err := client.Snapshots.Delete(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}
