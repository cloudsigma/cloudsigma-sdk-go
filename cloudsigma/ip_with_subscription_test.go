package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"
)

func TestIPWithSubscriptionPublic(t *testing.T) {
	data, err := os.ReadFile("ip_raw_public.json")
	if err != nil {
		t.Fatal(err)
	}
	f := &ipRawWire{t: t, data: data, code: 200}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	v, r, err := c.IPs.GetWithSubscription(context.Background(), "185.12.6.183")
	if err != nil || r == nil || r.StatusCode != 200 || f.calls != 1 {
		t.Fatal("transport", err)
	}
	if v.UUID != "185.12.6.183" || v.Gateway != "185.12.6.1" || v.Netmask != 24 || len(v.Nameservers) != 3 || v.Owner.UUID != "5b4a69a3-8e78-4c45-a8ba-8b13f0895e23" || v.Server != nil {
		t.Fatal("IP fields", v)
	}
	if v.Subscription == nil || v.Subscription.ID != 7273 || v.Subscription.ResourceURI != "/api/2.0/subscriptions/7273/" {
		t.Fatal("numeric reference", v)
	}
	v.Subscription.ID = 9
	v.Nameservers[0] = "changed"
	if v.Subscription.ID != 9 || v.Nameservers[0] != "changed" {
		t.Fatal("mutation")
	}
	again, _, err := c.IPs.GetWithSubscription(context.Background(), "185.12.6.183")
	if err != nil || again.Subscription.ID != 7273 || again.Nameservers[0] != "69.194.139.62" || f.calls != 2 {
		t.Fatal("fresh ownership", err)
	}
}

func TestIPWithSubscriptionPresence(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		nilRef     bool
		id         int
		uri        string
	}{
		{"absent", `{}`, true, 0, ""}, {"null", `{"subscription":null}`, true, 0, ""},
		{"empty", `{"subscription":{}}`, false, 0, ""},
		{"zero", `{"subscription":{"id":0}}`, false, 0, ""},
		{"null_fields", `{"subscription":{"id":null,"resource_uri":null}}`, false, 0, ""},
		{"missing_id", `{"subscription":{"resource_uri":"/x/"}}`, false, 0, "/x/"},
		{"extension", `{"subscription":{"id":7273,"extra":9007199254740993}}`, false, 7273, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &ipRawWire{t: t, data: []byte(tc.body), code: 200}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			v, r, e := c.IPs.GetWithSubscription(context.Background(), "185.12.6.183")
			if e != nil || r == nil || f.calls != 1 || (v.Subscription == nil) != tc.nilRef {
				t.Fatal("presence", e)
			}
			if !tc.nilRef && (v.Subscription.ID != tc.id || v.Subscription.ResourceURI != tc.uri) {
				t.Fatal("fields")
			}
		})
	}
}

func TestIPWithSubscriptionFailures(t *testing.T) {
	f := &ipRawWire{t: t, code: 200}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	for _, id := range []string{"", "..", "a/b", "a?b", "a#b", "%2f", "::1", "01.2.3.4", "1.2.3.999"} {
		v, r, e := c.IPs.GetWithSubscription(context.Background(), id)
		if e == nil || v != nil || r != nil || f.calls != 0 {
			t.Fatal("route refusal")
		}
	}
	for _, body := range []string{``, `{`, `[]`, `1`, `null`, `{"subscription":[]}`, `{"subscription":{"id":"7273"}}`, `{"subscription":{"id":1.5}}`, `{"subscription":{"id":true}}`, `{"subscription":{"resource_uri":3}}`} {
		f.data = []byte(body)
		v, r, e := c.IPs.GetWithSubscription(context.Background(), "185.12.6.183")
		if e == nil || v != nil || (r != nil) != (body == `null`) {
			t.Fatal("decoder", body, e)
		}
	}
	f.code = 404
	f.data = []byte(`[{"error_type":"notfound","error_message":"synthetic missing"}]`)
	v, r, e := c.IPs.GetWithSubscription(context.Background(), "185.12.6.183")
	var apiErr *ErrorResponse
	if v != nil || r == nil || r.StatusCode != 404 || !errors.As(e, &apiErr) || apiErr.Response != r || len(apiErr.Errors) != 1 || apiErr.Errors[0].Type != "notfound" || apiErr.Errors[0].Message != "synthetic missing" {
		t.Fatal("HTTP error", e)
	}
	f.err = errors.New("synthetic transport")
	v, r, e = c.IPs.GetWithSubscription(context.Background(), "185.12.6.183")
	if v != nil || r != nil || !errors.Is(e, f.err) {
		t.Fatal("transport error", e)
	}
	f.err = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	v, r, e = c.IPs.GetWithSubscription(ctx, "185.12.6.183")
	if v != nil || r != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("cancel", e)
	}
}

// Same numeric-reference oracle is also executed against legacy Get in the
// qualification baseline probe. This verifies actual typed field retention.
func TestIPWithSubscriptionRetentionOracle(t *testing.T) {
	f := &ipRawWire{t: t, code: 200, data: []byte(`{"uuid":"185.12.6.183","subscription":{"id":7273,"resource_uri":"/api/2.0/subscriptions/7273/"}}`)}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	v, _, e := c.IPs.GetWithSubscription(context.Background(), "185.12.6.183")
	if e != nil {
		t.Fatal(e)
	}
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	var got struct {
		Subscription *struct {
			ID          int
			ResourceURI string `json:"resource_uri"`
		}
	}
	if json.Unmarshal(b, &got) != nil || got.Subscription == nil || got.Subscription.ID != 7273 || got.Subscription.ResourceURI != "/api/2.0/subscriptions/7273/" {
		t.Fatal("typed reader drops documented numeric subscription reference", string(b))
	}
}
