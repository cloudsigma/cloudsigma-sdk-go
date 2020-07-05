package cloudsigma

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLibraryDrives_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/libdrives/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"arch":"64","uuid":"long-uuid"}],"meta":{"total_count":1}}`)
	})
	expected := []LibraryDrive{
		{
			Arch: "64",
			UUID: "long-uuid",
		},
	}

	drives, resp, err := client.LibraryDrives.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, drives)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestLibraryDrives_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/libdrives/long-uuid", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"arch":"64","uuid":"long-uuid"}`)
	})
	expected := &LibraryDrive{
		Arch: "64",
		UUID: "long-uuid",
	}

	libraryDrive, _, err := client.LibraryDrives.Get(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, libraryDrive)
}

func TestLibraryDrives_Get_emptyUUID(t *testing.T) {
	_, _, err := client.LibraryDrives.Get(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestLibraryDrives_Get_invalidUUID(t *testing.T) {
	_, _, err := client.LibraryDrives.Get(ctx, "%")

	assert.Error(t, err)
}

func TestLibraryDrives_Clone(t *testing.T) {
	setup()
	defer teardown()

	input := &LibraryDriveCloneRequest{
		LibraryDrive: &LibraryDrive{
			Size: 3221225472, // 3GB
		},
	}
	mux.HandleFunc("/libdrives/long-uuid/action/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "clone", r.URL.Query().Get("do"))
		_, _ = fmt.Fprint(w, `{"objects":[{"size":3221225472,"uuid":"generated-uuid"}]}`)
	})
	expected := &LibraryDrive{
		Size: 3221225472,
		UUID: "generated-uuid",
	}

	libraryDrive, _, err := client.LibraryDrives.Clone(ctx, "long-uuid", input)

	assert.NoError(t, err)
	assert.Equal(t, expected, libraryDrive)
}

func TestLibraryDrives_Clone_emptyPayload(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/libdrives/long-uuid/action/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "clone", r.URL.Query().Get("do"))
		_, _ = fmt.Fprint(w, `{"objects":[{"arch":"64","uuid":"generated-uuid"}]}`)
	})
	expected := &LibraryDrive{
		Arch: "64",
		UUID: "generated-uuid",
	}

	libraryDrive, _, err := client.LibraryDrives.Clone(ctx, "long-uuid", nil)

	assert.NoError(t, err)
	assert.Equal(t, expected, libraryDrive)
}

func TestLibraryDrives_Clone_emptyUUID(t *testing.T) {
	input := &LibraryDriveCloneRequest{
		LibraryDrive: &LibraryDrive{
			Size: 3221225472, // 3GB
		},
	}

	_, _, err := client.LibraryDrives.Clone(ctx, "", input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestLibraryDrives_Clone_invalidUUID(t *testing.T) {
	input := &LibraryDriveCloneRequest{
		LibraryDrive: &LibraryDrive{
			Size: 3221225472, // 3GB
		},
	}

	_, _, err := client.LibraryDrives.Clone(ctx, "%", input)

	assert.Error(t, err)
}
