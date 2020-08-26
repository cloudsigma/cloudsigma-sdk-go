package cloudsigma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoteSnapshots_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/remotesnapshots/detail/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"uuid":"long-uuid","location":"ZRH"}],"meta":{"total_count":1}}`)
	})
	expected := []RemoteSnapshot{
		{
			Location: "ZRH",
			Snapshot: Snapshot{UUID: "long-uuid"},
		},
	}

	remoteSnapshots, resp, err := client.RemoteSnapshots.List(ctx, nil)

	assert.NoError(t, err)
	assert.Equal(t, expected, remoteSnapshots)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestRemoteSnapshots_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/remotesnapshots/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"meta":{"key":"value"},"uuid":"long-uuid"}`)
	})
	expected := &RemoteSnapshot{
		Snapshot: Snapshot{
			Meta: map[string]interface{}{
				"key": "value",
			},
			UUID: "long-uuid",
		},
	}

	remoteSnapshot, _, err := client.RemoteSnapshots.Get(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, remoteSnapshot)
}

func TestRemoteSnapshots_Get_emptyUUID(t *testing.T) {
	_, _, err := client.RemoteSnapshots.Get(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestRemoteSnapshots_Get_invalidUUID(t *testing.T) {
	_, _, err := client.RemoteSnapshots.Get(ctx, "%")

	assert.Error(t, err)
}

func TestRemoteSnapshots_Create(t *testing.T) {
	setup()
	defer teardown()

	input := &RemoteSnapshotCreateRequest{
		RemoteSnapshots: []RemoteSnapshot{
			{
				Snapshot: Snapshot{Name: "test remote snapshot"},
			},
		},
	}
	mux.HandleFunc("/remotesnapshots/", func(w http.ResponseWriter, r *http.Request) {
		v := new(RemoteSnapshotCreateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test remote snapshot","uuid":"long-uuid"}]}`)
	})
	expected := []RemoteSnapshot{
		{Snapshot: Snapshot{Name: "test remote snapshot", UUID: "long-uuid"}},
	}

	remoteSnapshots, _, err := client.RemoteSnapshots.Create(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, expected, remoteSnapshots)
}

func TestRemoteSnapshots_Create_emptyPayload(t *testing.T) {
	_, _, err := client.RemoteSnapshots.Create(ctx, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestRemoteSnapshots_Update(t *testing.T) {
	setup()
	defer teardown()

	input := &RemoteSnapshotUpdateRequest{
		RemoteSnapshot: &RemoteSnapshot{
			Snapshot: Snapshot{
				Name: "test remote snapshot v2",
			},
		},
	}
	mux.HandleFunc("/remotesnapshots/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		v := new(RemoteSnapshotUpdateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPut, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test remote snapshot v2","uuid":"long-uuid"}`)
	})
	expected := &RemoteSnapshot{
		Snapshot: Snapshot{
			Name: "test remote snapshot v2",
			UUID: "long-uuid",
		},
	}

	remoteSnapshot, _, err := client.RemoteSnapshots.Update(ctx, "long-uuid", input)

	assert.NoError(t, err)
	assert.Equal(t, expected, remoteSnapshot)
}

func TestRemoteSnapshots_Update_emptyUUID(t *testing.T) {
	input := &RemoteSnapshotUpdateRequest{
		RemoteSnapshot: &RemoteSnapshot{
			Snapshot: Snapshot{
				Name: "test remote snapshot v2",
			},
		},
	}

	_, _, err := client.RemoteSnapshots.Update(ctx, "", input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestRemoteSnapshots_Update_invalidUUID(t *testing.T) {
	input := &RemoteSnapshotUpdateRequest{
		RemoteSnapshot: &RemoteSnapshot{
			Snapshot: Snapshot{
				Name: "test remote snapshot v2",
			},
		},
	}

	_, _, err := client.RemoteSnapshots.Update(ctx, "%", input)

	assert.Error(t, err)
}

func TestRemoteSnapshots_Update_emptyPayload(t *testing.T) {
	_, _, err := client.RemoteSnapshots.Update(ctx, "long-uuid", nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestRemoteSnapshots_Delete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/remotesnapshots/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
	})

	_, err := client.RemoteSnapshots.Delete(ctx, "long-uuid")

	assert.NoError(t, err)
}

func TestRemoteSnapshots_Delete_emptyUUID(t *testing.T) {
	_, err := client.RemoteSnapshots.Delete(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}
