package cloudsigma_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// This is a public-documentation acknowledgment fixture, not a lifecycle test.
func TestACLCreateDefinitionPublicContract(t *testing.T) {
	for _, mode := range []string{"empty", "grantees"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			c := sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
				calls++
				b, err := io.ReadAll(q.Body)
				if err != nil {
					t.Fatal(err)
				}
				_ = q.Body.Close()
				if q.Method != "POST" || q.URL.RequestURI() != "/custom/api/2.0/acls/" || q.Header.Get("Authorization") != "Bearer synthetic" {
					t.Fatalf("request %s %s auth=%q", q.Method, q.URL, q.Header.Get("Authorization"))
				}
				var got, want interface{}
				if json.Unmarshal(b, &got) != nil || json.Unmarshal([]byte(aclcreatePreparseBody(mode)), &want) != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("body %s", b)
				}
				return &http.Response{StatusCode: 201, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"name":"fixture","uuid":"adf048e8-5445-47cf-a09c-fc3cfb2c823f","grantees":[],"rules":[],"tags":[]}]}`)), Request: q}, nil
			})}))
			object := sdk.ACLDefinition{"name": json.RawMessage(`"fixture"`), "grantees": json.RawMessage(`[]`), "rules": json.RawMessage(`[]`), "tags": json.RawMessage(`[]`)}
			if mode == "grantees" {
				object["grantees"] = json.RawMessage(`[{"email":"user@example.com","uuid":"c2fc9982-cf2e-434a-bf63-e22a27b39f00"}]`)
				object["rules"] = json.RawMessage(`[{"permission":"LIST"}]`)
			}
			out, r, e := c.ACLs.CreateDefinition(context.Background(), []sdk.ACLDefinition{object})
			if e != nil || r == nil || r.StatusCode != 201 || calls != 1 || !strings.Contains(string(out["objects"]), "adf048e8-5445-47cf-a09c-fc3cfb2c823f") {
				t.Fatalf("ACK %v %v %v", out, r, e)
			}
			out, r, e = c.ACLs.CreateDefinition(context.Background(), nil)
			if out != nil || r != nil || !errors.Is(e, sdk.ErrEmptyPayloadNotAllowed) || calls != 1 {
				t.Fatal("nil payload dispatched")
			}
			for _, bad := range []sdk.ACLDefinition{nil, {"uuid": json.RawMessage(`null`)}, {"name": json.RawMessage(`invalid`)}} {
				out, r, e = c.ACLs.CreateDefinition(context.Background(), []sdk.ACLDefinition{bad})
				if out != nil || r != nil || e == nil || calls != 1 {
					t.Fatal("invalid definition dispatched")
				}
			}
		})
	}
}
