package cloudsigma

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPs_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/ips/185.12.6.243", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"gateway":"185.12.6.1","uuid":"185.12.6.243"}`)
	})
	expected := &IP{
		Gateway: "185.12.6.1",
		UUID:    "185.12.6.243",
	}

	ip, _, err := client.IPs.Get(ctx, "185.12.6.243")

	assert.NoError(t, err)
	assert.Equal(t, expected, ip)
}

func TestIPs_Get_emptyUUID(t *testing.T) {
	_, _, err := client.IPs.Get(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestIPs_Get_invalidUUID(t *testing.T) {
	_, _, err := client.IPs.Get(ctx, "%")

	assert.Error(t, err)
}
