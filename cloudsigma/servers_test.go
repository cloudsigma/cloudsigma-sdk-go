package cloudsigma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServers_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/servers/detail/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test server","uuid":"long-uuid"}],"meta":{"total_count":1}}`)
	})
	expected := []Server{
		{
			Name: "test server",
			UUID: "long-uuid",
		},
	}

	servers, resp, err := client.Servers.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, servers)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestServers_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/servers/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test server","uuid":"long-uuid"}`)
	})
	expected := &Server{
		Name: "test server",
		UUID: "long-uuid",
	}

	server, _, err := client.Servers.Get(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, server)
}

func TestServers_Get_emptyUUID(t *testing.T) {
	_, _, err := client.Servers.Get(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestServers_Get_invalidUUID(t *testing.T) {
	_, _, err := client.Servers.Get(ctx, "%")

	assert.Error(t, err)
}

func TestServers_Create(t *testing.T) {
	setup()
	defer teardown()

	input := &ServerCreateRequest{
		Servers: []Server{
			{Name: "test server"},
		},
	}
	mux.HandleFunc("/servers/", func(w http.ResponseWriter, r *http.Request) {
		v := new(ServerCreateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"name":"test server","uuid":"long-uuid"}]}`)
	})
	expected := []Server{
		{Name: "test server", UUID: "long-uuid"},
	}

	servers, _, err := client.Servers.Create(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, expected, servers)
}

func TestServers_Create_emptyPayload(t *testing.T) {
	_, _, err := client.Servers.Create(ctx, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestServers_Update(t *testing.T) {
	setup()
	defer teardown()

	input := &ServerUpdateRequest{
		Server: &Server{
			Name: "test server v2",
		},
	}
	mux.HandleFunc("/servers/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		v := new(ServerUpdateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPut, r.Method)
		_, _ = fmt.Fprint(w, `{"name":"test server v2","uuid":"long-uuid"}`)
	})
	expected := &Server{
		Name: "test server v2",
		UUID: "long-uuid",
	}

	server, _, err := client.Servers.Update(ctx, "long-uuid", input)

	assert.NoError(t, err)
	assert.Equal(t, expected, server)
}

func TestServers_Update_emptyUUID(t *testing.T) {
	input := &ServerUpdateRequest{
		Server: &Server{
			Name: "test server v2",
		},
	}

	_, _, err := client.Servers.Update(ctx, "", input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestServers_Update_invalidUUID(t *testing.T) {
	input := &ServerUpdateRequest{
		Server: &Server{
			Name: "test server v2",
		},
	}

	_, _, err := client.Servers.Update(ctx, "%", input)

	assert.Error(t, err)
}

func TestServers_Update_emptyPayload(t *testing.T) {
	_, _, err := client.Servers.Update(ctx, "long-uuid", nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}

func TestServers_Delete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/servers/long-uuid/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
	})

	_, err := client.Servers.Delete(ctx, "long-uuid")

	assert.NoError(t, err)
}

func TestServer_Delete_emptyUUID(t *testing.T) {
	_, err := client.Servers.Delete(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestServer_Start(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/servers/long-uuid/action/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "start", r.URL.Query().Get("do"))
		_, _ = fmt.Fprint(w, `{"action":"start","result":"success","uuid":"long-uuid"}`)
	})
	expected := &ServerAction{
		Action: "start",
		Result: "success",
		UUID:   "long-uuid",
	}

	action, _, err := client.Servers.Start(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, action)
}

func TestServer_Start_emptyUUID(t *testing.T) {
	_, _, err := client.Servers.Start(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestServer_Stop(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/servers/long-uuid/action/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "stop", r.URL.Query().Get("do"))
		_, _ = fmt.Fprint(w, `{"action":"stop","result":"success","uuid":"long-uuid"}`)
	})
	expected := &ServerAction{
		Action: "stop",
		Result: "success",
		UUID:   "long-uuid",
	}

	action, _, err := client.Servers.Stop(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, action)
}

func TestServer_Stop_emptyUUID(t *testing.T) {
	_, _, err := client.Servers.Stop(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}

func TestServer_Shutdown(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/servers/long-uuid/action/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "shutdown", r.URL.Query().Get("do"))
		_, _ = fmt.Fprint(w, `{"action":"shutdown","result":"success","uuid":"long-uuid"}`)
	})
	expected := &ServerAction{
		Action: "shutdown",
		Result: "success",
		UUID:   "long-uuid",
	}

	action, _, err := client.Servers.Shutdown(ctx, "long-uuid")

	assert.NoError(t, err)
	assert.Equal(t, expected, action)
}

func TestServer_Shutdown_emptyUUID(t *testing.T) {
	_, _, err := client.Servers.Shutdown(ctx, "")

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyArgument.Error(), err.Error())
}
