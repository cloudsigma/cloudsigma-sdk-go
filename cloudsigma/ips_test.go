package cloudsigma

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPs_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/ips/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"gateway":"185.12.6.1","uuid":"185.12.6.243"}],"meta":{"total_count":1}}`)
	})
	expected := []IP{
		{
			Gateway: "185.12.6.1",
			UUID:    "185.12.6.243",
		},
	}

	ips, resp, err := client.IPs.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, ips)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestIPs_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/ips/185.12.6.243/", func(w http.ResponseWriter, r *http.Request) {
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
