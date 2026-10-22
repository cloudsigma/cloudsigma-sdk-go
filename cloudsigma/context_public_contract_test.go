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

// Literal expectations follow the public global-context POST example.
// A decoded acknowledgment does not establish propagation into guest contexts.
func TestContextPreparsePublicContract(t *testing.T) {
	for _, code := range []int{200, 201, 202} {
		calls := 0
		input := map[string]string{"new_global_key": "new_global_val"}
		c := sdk.NewClient(sdk.NewTokenCredentialsProvider("contract-token"), sdk.WithEndpoint("https://example.invalid/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
			calls++
			if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/global_context/" || q.Header.Get("Authorization") != "Bearer contract-token" {
				t.Error("public route/auth changed")
			}
			var body map[string]string
			if e := json.NewDecoder(q.Body).Decode(&body); e != nil {
				t.Fatal(e)
			}
			_ = q.Body.Close()
			if !reflect.DeepEqual(body, map[string]string{"new_global_key": "new_global_val"}) {
				t.Fatalf("documented payload changed: %#v", body)
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"new_global_key":"new_global_val","large":9007199254740993,"nullable":null}`)), Request: q}, nil
		})}))
		v, r, e := c.GlobalContext.Update(context.Background(), input)
		if e != nil || r == nil || r.StatusCode != code || v == nil || calls != 1 || !reflect.DeepEqual(input, map[string]string{"new_global_key": "new_global_val"}) {
			t.Fatalf("acknowledgment changed: %v %v %v calls=%d", v, r, e, calls)
		}
		if string((*v)["new_global_key"]) != `"new_global_val"` || string((*v)["large"]) != "9007199254740993" || string((*v)["nullable"]) != "null" {
			t.Fatal("raw context acknowledgment changed")
		}
	}
}
