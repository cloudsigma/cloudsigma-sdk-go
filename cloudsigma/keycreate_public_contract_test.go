package cloudsigma_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// Literal independent oracle for the documented imported-key POST contract.
// HTTP201 acknowledges the fixture only; it proves no cloud key usability.
func TestKeycreatePublicContract(t *testing.T) {
	for _, both := range []bool{false, true} {
		calls := 0
		key := sdk.Keypair{Name: "imported", PublicKey: "ssh-rsa SYNTHETIC"}
		want := `{"objects":[{"name":"imported","public_key":"ssh-rsa SYNTHETIC"}]}`
		if both {
			key.PrivateKey = "SYNTHETIC-PRIVATE"
			want = `{"objects":[{"name":"imported","private_key":"SYNTHETIC-PRIVATE","public_key":"ssh-rsa SYNTHETIC"}]}`
		}
		c := sdk.NewClient(sdk.NewTokenCredentialsProvider("fixture-token"), sdk.WithEndpoint("https://example.invalid/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
			calls++
			b, _ := io.ReadAll(q.Body)
			_ = q.Body.Close()
			if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/keypairs/" || strings.TrimSpace(string(b)) != want || q.Header.Get("Authorization") != "Bearer fixture-token" {
				t.Errorf("literal route/body/auth mismatch: %s %s %s auth=%q", q.Method, q.URL, b, q.Header.Get("Authorization"))
			}
			return &http.Response{StatusCode: 201, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"ack-id","name":"imported","public_key":"ssh-rsa SYNTHETIC","fingerprint":"fixture-fingerprint"}]}`)), Request: q}, nil
		})}))
		v, r, e := c.Keypairs.Create(context.Background(), &sdk.KeypairCreateRequest{Keypairs: []sdk.Keypair{key}})
		if e != nil || r == nil || r.StatusCode != 201 || len(v) != 1 || v[0].UUID != "ack-id" || v[0].Fingerprint != "fixture-fingerprint" || calls != 1 {
			t.Fatalf("ACK decoding %v %v %v calls=%d", v, r, e, calls)
		}
		v, r, e = c.Keypairs.Create(context.Background(), nil)
		if v != nil || r != nil || e != sdk.ErrEmptyPayloadNotAllowed || calls != 1 {
			t.Fatal("nil payload validation changed")
		}
	}
}
