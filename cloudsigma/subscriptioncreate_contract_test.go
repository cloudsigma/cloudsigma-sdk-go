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

// Public POST route and 201 acknowledgment fixture, not purchase/lifecycle proof.
// The public schema declares amount a string although its example uses a number.
// False auto_renew omission is legacy behavior, not full schema fidelity.
func TestSubscriptionCreatePublicContract(t *testing.T) {
	for _, mode := range []string{"single", "renew"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			c := sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
				calls++
				b, err := io.ReadAll(q.Body)
				if err != nil {
					t.Fatal(err)
				}
				_ = q.Body.Close()
				if q.Method != "POST" || q.URL.RequestURI() != "/custom/api/2.0/subscriptions/" || q.Header.Get("Authorization") != "Bearer synthetic" {
					t.Fatalf("request %s %s auth=%q", q.Method, q.URL, q.Header.Get("Authorization"))
				}
				var got, want interface{}
				if json.Unmarshal(b, &got) != nil || json.Unmarshal([]byte(subscriptioncreatePreparseBody(mode)), &want) != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("body %s", b)
				}
				return &http.Response{StatusCode: 201, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"amount":"30000","resource":"dssd","uuid":"adf048e8-5445-47cf-a09c-fc3cfb2c823f"}]}`)), Request: q}, nil
			})}))
			object := sdk.Subscription{Amount: "30000", Period: "1 month", Resource: "dssd"}
			if mode == "renew" {
				object.AutoRenew = true
			}
			out, r, e := c.Subscriptions.Create(context.Background(), &sdk.SubscriptionCreateRequest{Subscriptions: []sdk.Subscription{object}})
			if e != nil || r == nil || r.StatusCode != 201 || calls != 1 || len(out) != 1 || out[0].UUID != "adf048e8-5445-47cf-a09c-fc3cfb2c823f" {
				t.Fatalf("ACK %v %v %v", out, r, e)
			}
			out, r, e = c.Subscriptions.Create(context.Background(), nil)
			if out != nil || r != nil || !errors.Is(e, sdk.ErrEmptyPayloadNotAllowed) || calls != 1 {
				t.Fatal("nil payload dispatched")
			}
		})
	}
}
