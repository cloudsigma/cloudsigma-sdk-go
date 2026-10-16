package cloudsigma

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"testing"
)

// Characterizes the existing escape hatch, not a new typed IP response API.
func TestIPSubscriptionExistingReaderFidelity(t *testing.T) {
	data, err := os.ReadFile("ip_raw_public.json")
	if err != nil {
		t.Fatal(err)
	}
	f := &ipRawWire{t: t, data: data, code: 200}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	legacy, _, err := c.IPs.Get(context.Background(), "185.12.6.183")
	if err != nil || legacy.UUID != "185.12.6.183" {
		t.Fatal("legacy", err)
	}
	if _, ok := reflect.TypeOf(*legacy).FieldByName("Subscription"); ok {
		t.Fatal("reassess legacy projection: field now exists")
	}
	raw, response, err := c.IPs.GetRaw(context.Background(), "185.12.6.183")
	var subscription VLANSubscription
	if err != nil || response == nil || response.StatusCode != 200 || f.calls != 2 {
		t.Fatal("read", err)
	}
	if err = json.Unmarshal(raw["subscription"], &subscription); err != nil {
		t.Fatal(err)
	}
	if subscription.ID != 7273 || subscription.ResourceURI != "/api/2.0/subscriptions/7273/" {
		t.Fatal(subscription)
	}
	// The full Subscription type has a string ID and is NOT this reference shape.
	var full Subscription
	if json.Unmarshal(raw["subscription"], &full) == nil {
		t.Fatal("numeric reference unexpectedly fits full subscription")
	}
	subscription.ID = 1
	if subscription.ID != 1 {
		t.Fatal("mutation")
	}
	var again VLANSubscription
	if err = json.Unmarshal(raw["subscription"], &again); err != nil || again.ID != 7273 {
		t.Fatal("typed decode aliases raw")
	}
}

func TestIPSubscriptionRawPresence(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		present          bool
	}{
		{"absent", `{}`, ``, false},
		{"null", `{"subscription":null}`, `null`, true},
		{"empty", `{"subscription":{}}`, `{}`, true},
		{"zero", `{"subscription":{"id":0}}`, `{"id":0}`, true},
		{"missing_id", `{"subscription":{"resource_uri":"/x/"}}`, `{"resource_uri":"/x/"}`, true},
		{"null_id", `{"subscription":{"id":null}}`, `{"id":null}`, true},
		{"extension", `{"subscription":{"id":7273,"extra":9007199254740993}}`, `{"id":7273,"extra":9007199254740993}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &ipRawWire{t: t, data: []byte(tc.body), code: 200}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			raw, r, err := c.IPs.GetRaw(context.Background(), "185.12.6.183")
			value, present := raw["subscription"]
			if err != nil || r == nil || f.calls != 1 || present != tc.present || string(value) != tc.want {
				t.Fatal("presence", raw, err)
			}
			if len(value) > 0 {
				value[0] = 'X'
				if raw["subscription"][0] != 'X' {
					t.Fatal("mutation did not occur")
				}
				next, _, e := c.IPs.GetRaw(context.Background(), "185.12.6.183")
				if e != nil || string(next["subscription"]) != tc.want || f.calls != 2 {
					t.Fatal("repeated response ownership")
				}
			}
		})
	}
}
