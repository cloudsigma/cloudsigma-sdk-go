package cloudsigma

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDrives_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/drives/long-uuid", func(w http.ResponseWriter, r *http.Request) {
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
