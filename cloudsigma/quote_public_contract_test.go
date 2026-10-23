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

// Public calculator examples describe quotations, not purchase or extension.
func TestQuotePreparsePublicContract(t *testing.T) {
	for _, action := range []string{"calculate", "extend"} {
		t.Run(action, func(t *testing.T) {
			calls := 0
			c := sdk.NewClient(sdk.NewTokenCredentialsProvider("contract-token"), sdk.WithEndpoint("https://example.invalid/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
				calls++
				path := "/api/2.0/subscriptioncalculator/"
				expected := `{"objects":[{"amount":10000000000,"period":"1 month","resource":"dssd"}]}`
				if action == "extend" {
					path = "/api/2.0/subscriptioncalculator/7272/action/?do=extend"
					expected = `{}`
				}
				if q.Method != "POST" || q.URL.RequestURI() != path || q.Header.Get("Authorization") != "Bearer contract-token" {
					t.Error("public route/auth changed")
				}
				b, e := io.ReadAll(q.Body)
				if e != nil {
					t.Fatal(e)
				}
				_ = q.Body.Close()
				var got, want interface{}
				d := json.NewDecoder(strings.NewReader(string(b)))
				d.UseNumber()
				if e = d.Decode(&got); e != nil {
					t.Fatal(e)
				}
				d = json.NewDecoder(strings.NewReader(expected))
				d.UseNumber()
				if e = d.Decode(&want); e != nil {
					t.Fatal(e)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("payload changed: %s", b)
				}
				payload := `{"objects":[{"amount":"10000000000","resource":"dssd","price":"1.010484993457794189453125"}],"price":"1.010484993457794189453125","large":9007199254740993,"nullable":null}`
				if action == "extend" {
					payload = `{"amount":"32212254720","discount_amount":"9.60750000000000000","discount_percent":"0.2500000000","discounts":{"period":"0.2500000000"},"period":"366 days, 0:00:00","price":"28.82250000000000000000","resource":"dssd","large":9007199254740993,"nullable":null}`
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload)), Request: q}, nil
			})}))
			var v map[string]json.RawMessage
			var r *sdk.Response
			var e error
			if action == "extend" {
				v, r, e = c.Subscriptions.CalculateExtension(context.Background(), "7272")
			} else {
				v, r, e = c.Subscriptions.Calculate(context.Background(), []json.RawMessage{json.RawMessage(`{"amount":10000000000,"period":"1 month","resource":"dssd"}`)})
			}
			price := `"1.010484993457794189453125"`
			if action == "extend" {
				price = `"28.82250000000000000000"`
			}
			if e != nil || r == nil || r.StatusCode != 200 || calls != 1 || string(v["price"]) != price || string(v["large"]) != "9007199254740993" || string(v["nullable"]) != "null" {
				t.Fatalf("quote changed: %v %v %v calls=%d", v, r, e, calls)
			}
		})
	}
}
