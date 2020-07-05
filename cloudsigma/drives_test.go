package cloudsigma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDrives_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/drives/detail/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test drive","uuid":"long-uuid"}],"meta":{"total_count":1}}`)
	})
	expected := []Drive{
		{
			Name: "test drive",
			UUID: "long-uuid",
		},
	}

	drives, resp, err := client.Drives.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, drives)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestDrives_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/drives/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test drive","uuid":"long-uuid"}`)
	})
	expected := &Drive{
		Name: "test drive",
		UUID: "long-uuid",
	}

	drive, _, err := client.Drives.Get(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, drive)
}

func TestDrives_Get_emptyUUID(t *testing.T) {
	_, _, err := client.Drives.Get(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestDrives_Get_invalidUUID(t *testing.T) {
	_, _, err := client.Drives.Get(ctx, "%")

	assert.Error(t, err)
}

func TestDrives_Create(t *testing.T) {
	setup()
	defer teardown()

	input := &DriveCreateRequest{
		Drives: []Drive{
			{Name: "test drive"},
		},
	}
	mux.HandleFunc("/drives/", func(w http.ResponseWriter, r *http.Request) {
		v := new(DriveCreateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test drive","uuid":"long-uuid"}]}`)
	})
	expected := []Drive{
		{Name: "test drive", UUID: "long-uuid"},
	}

	drives, _, err := client.Drives.Create(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, expected, drives)
}

func TestDrives_Create_emptyPayload(t *testing.T) {
	_, _, err := client.Drives.Create(ctx, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestDrives_Update(t *testing.T) {
	setup()
	defer teardown()

	input := &DriveUpdateRequest{
		Drive: &Drive{
			Name: "test drive v2",
		},
	}
	mux.HandleFunc("/drives/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		v := new(DriveUpdateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPut, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test drive v2","uuid":"long-uuid"}`)
	})
	expected := &Drive{
		Name: "test drive v2",
		UUID: "long-uuid",
	}

	drive, _, err := client.Drives.Update(ctx, "long-uuid", input)

	assert.NoError(t, err)
	assert.Equal(t, expected, drive)
}

func TestDrives_Update_emptyUUID(t *testing.T) {
	input := &DriveUpdateRequest{
		Drive: &Drive{
			Name: "test drive v2",
		},
	}

	_, _, err := client.Drives.Update(ctx, "", input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestDrives_Update_invalidUUID(t *testing.T) {
	input := &DriveUpdateRequest{
		Drive: &Drive{
			Name: "test drive v2",
		},
	}

	_, _, err := client.Drives.Update(ctx, "%", input)

	assert.Error(t, err)
}

func TestDrives_Update_emptyPayload(t *testing.T) {
	_, _, err := client.Drives.Update(ctx, "long-uuid", nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestDrives_Delete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/drives/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
	})

	_, err := client.Drives.Delete(ctx, "long-uuid")

	assert.NoError(t, err)
}

func TestDrives_Delete_emptyUUID(t *testing.T) {
	_, err := client.Drives.Delete(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestDrives_Resize(t *testing.T) {
	setup()
	defer teardown()

	input := &DriveUpdateRequest{
		Drive: &Drive{
			Name: "test drive v2",
		},
	}
	mux.HandleFunc("/drives/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		v := new(DriveUpdateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test drive v2","uuid":"long-uuid"}]}`)
	})
	expected := []Drive{
		{Name: "test drive v2", UUID: "long-uuid"},
	}

	drives, _, err := client.Drives.Resize(ctx, "long-uuid", input)

	assert.NoError(t, err)
	assert.Equal(t, expected, drives)
}

func TestDrives_Resize_emptyUUID(t *testing.T) {
	input := &DriveUpdateRequest{
		Drive: &Drive{
			Name: "test drive v2",
		},
	}

	_, _, err := client.Drives.Resize(ctx, "", input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestDrives_Resize_invalidUUID(t *testing.T) {
	input := &DriveUpdateRequest{
		Drive: &Drive{
			Name: "test drive v2",
		},
	}

	_, _, err := client.Drives.Resize(ctx, "%", input)

	assert.Error(t, err)
}

func TestDrives_Resize_emptyPayload(t *testing.T) {
	_, _, err := client.Drives.Resize(ctx, "long-uuid", nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestDrives_Clone(t *testing.T) {
	setup()
	defer teardown()

	input := &DriveCloneRequest{
		Drive: &Drive{
			Size: 3221225472, // 3GB
		},
	}
	mux.HandleFunc("/drives/long-uuid/action/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "clone", r.URL.Query().Get("do"))
		_, _ = fmt.Fprint(w, `{"objects":[{"size":3221225472,"uuid":"generated-uuid"}]}`)
	})
	expected := &Drive{
		Size: 3221225472,
		UUID: "generated-uuid",
	}

	drive, _, err := client.Drives.Clone(ctx, "long-uuid", input)

	assert.NoError(t, err)
	assert.Equal(t, expected, drive)
}

func TestDrives_Clone_emptyPayload(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/drives/long-uuid/action/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "clone", r.URL.Query().Get("do"))
		_, _ = fmt.Fprint(w, `{"objects":[{"uuid":"generated-uuid"}]}`)
	})
	expected := &Drive{
		UUID: "generated-uuid",
	}

	drive, _, err := client.Drives.Clone(ctx, "long-uuid", nil)

	assert.NoError(t, err)
	assert.Equal(t, expected, drive)
}

func TestDrives_Clone_emptyUUID(t *testing.T) {
	input := &DriveCloneRequest{
		Drive: &Drive{
			Size: 3221225472, // 3GB
		},
	}

	_, _, err := client.Drives.Clone(ctx, "", input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestDrives_Clone_invalidUUID(t *testing.T) {
	input := &DriveCloneRequest{
		Drive: &Drive{
			Size: 3221225472, // 3GB
		},
	}

	_, _, err := client.Drives.Clone(ctx, "&", input)

	assert.Error(t, err)
}
