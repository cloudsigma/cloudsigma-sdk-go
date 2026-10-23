package cloudsigma_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// Literal expectations follow the public RDS drive user-count POST example.
// HTTP acknowledgment is not provisioning, mounting or remote lifecycle proof.
func TestDriveUserPreparsePublicContract(t *testing.T) {
	for _, code := range []int{201, 200, 202} {
		calls := 0
		c := sdk.NewClient(sdk.NewTokenCredentialsProvider("contract-token"), sdk.WithEndpoint("https://example.invalid/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
			calls++
			if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/driveusers/" || q.Header.Get("Authorization") != "Bearer contract-token" {
				t.Error("public route/auth changed")
			}
			var body map[string]json.RawMessage
			if e := json.NewDecoder(q.Body).Decode(&body); e != nil {
				t.Fatal(e)
			}
			_ = q.Body.Close()
			if !reflect.DeepEqual(body, map[string]json.RawMessage{"drive": json.RawMessage(`"e26fd531-7336-40e7-9a6f-db42e7a6b9f2"`), "amount": json.RawMessage(`5`)}) {
				t.Fatalf("documented payload changed: %#v", body)
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"amount":5,"drive":{"resource_uri":"/api/2.0/drives/e26fd531-7336-40e7-9a6f-db42e7a6b9f2/","uuid":"e26fd531-7336-40e7-9a6f-db42e7a6b9f2"}}],"large":9007199254740993,"nullable":null}`)), Request: q}, nil
		})}))
		v, r, e := c.DriveUsers.Create(context.Background(), "e26fd531-7336-40e7-9a6f-db42e7a6b9f2", 5)
		if e != nil || r == nil || r.StatusCode != code || v == nil || calls != 1 {
			t.Fatalf("acknowledgment changed: %v %v %v calls=%d", v, r, e, calls)
		}
		if string(v["objects"]) != `[{"amount":5,"drive":{"resource_uri":"/api/2.0/drives/e26fd531-7336-40e7-9a6f-db42e7a6b9f2/","uuid":"e26fd531-7336-40e7-9a6f-db42e7a6b9f2"}}]` || string(v["large"]) != "9007199254740993" || string(v["nullable"]) != "null" {
			t.Fatal("raw create acknowledgment changed")
		}
	}
}
