package cloudsigma_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// The public resize example requires a full definition. 202 is an ACK, not
// completed resizing. Token auth is existing client compatibility only.
func TestResizeDefinitionPublicContract(t *testing.T) {
	raw, err := os.ReadFile("resize_public_request.json")
	if err != nil {
		t.Fatal(err)
	}
	var definition sdk.DriveDefinition
	if err = json.Unmarshal(raw, &definition); err != nil {
		t.Fatal(err)
	}
	calls := 0
	c := sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
		calls++
		b, e := io.ReadAll(q.Body)
		if e != nil {
			t.Fatal(e)
		}
		_ = q.Body.Close()
		if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/drives/fixture/action/?do=resize" || q.Header.Get("Authorization") != "Bearer synthetic" {
			t.Fatal("route/auth changed")
		}
		var got, want sdk.DriveDefinition
		if json.Unmarshal(b, &got) != nil || json.Unmarshal(raw, &want) != nil {
			t.Fatal("invalid JSON")
		}
		// Compare each raw JSON value structurally with number lexemes preserved.
		decode := func(b []byte) interface{} {
			d := json.NewDecoder(strings.NewReader(string(b)))
			d.UseNumber()
			var v interface{}
			if e := d.Decode(&v); e != nil {
				t.Fatal(e)
			}
			return v
		}
		if !reflect.DeepEqual(decode(b), decode(raw)) {
			t.Fatal("full definition changed")
		}
		return &http.Response{StatusCode: 202, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"fixture","status":"resizing"}]}`)), Request: q}, nil
	})}))
	out, r, e := c.Drives.ResizeDefinition(context.Background(), "fixture", definition)
	if e != nil || r == nil || r.StatusCode != 202 || calls != 1 || len(out) != 1 || out[0].UUID != "fixture" || out[0].Status != "resizing" {
		t.Fatalf("ACK %v %v %v", out, r, e)
	}
	for _, id := range []string{"", "../unsafe", "id?do=delete"} {
		out, r, e = c.Drives.ResizeDefinition(context.Background(), id, definition)
		if e == nil || out != nil || r != nil || calls != 1 {
			t.Fatal("invalid identifier dispatched")
		}
	}
	out, r, e = c.Drives.ResizeDefinition(context.Background(), "fixture", nil)
	if !errors.Is(e, sdk.ErrEmptyPayloadNotAllowed) || out != nil || r != nil || calls != 1 {
		t.Fatal("nil definition dispatched")
	}
	out, r, e = c.Drives.ResizeDefinition(context.Background(), "fixture", sdk.DriveDefinition{"bad": []byte(`{`)})
	if e == nil || out != nil || r != nil || calls != 1 {
		t.Fatal("invalid JSON dispatched")
	}
}
