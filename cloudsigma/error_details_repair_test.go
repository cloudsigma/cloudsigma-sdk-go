package cloudsigma

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestIndependentMissingResponseRequest(t *testing.T) {
	ctx := WithErrorDetails(context.Background())
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(`[{"error_point":null}]`)), Header: make(http.Header)}, nil
	})}))
	_, _, err := c.Balance.Get(ctx)
	if err == nil {
		t.Fatal("missing legacy error")
	}
	if ErrorDetailsFromContext(ctx) == nil {
		t.Fatal("opt-in actual SDK error body lost when valid RoundTripper omits optional Response.Request")
	}
}
func TestErrorDetailsUnsupportedReuseBeforeDispatch(t *testing.T) {
	ctx := WithErrorDetails(context.Background())
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 400, Request: q, Body: io.NopCloser(strings.NewReader(`[{"error_point":null}]`)), Header: make(http.Header)}, nil
	})}))
	_, _, _ = c.Balance.Get(ctx)
	if ErrorDetailsFromContext(ctx) == nil {
		t.Fatal("setup")
	}
	_, _, err := c.Servers.OpenVNC(ctx, "")
	if err == nil {
		t.Fatal("expected validation failure")
	}
	if ErrorDetailsFromContext(ctx) == nil {
		t.Fatal("unsupported reuse characterization changed")
	}
}

func TestErrorDetailsUnrelatedResponseRequest(t *testing.T) {
	ctx := WithErrorDetails(context.Background())
	other := WithErrorDetails(context.Background())
	unrelated, _ := http.NewRequestWithContext(other, "GET", "https://example.invalid/", nil)
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 400, Request: unrelated, Body: io.NopCloser(strings.NewReader(`[{"error_point":null}]`)), Header: make(http.Header)}, nil
	})}))
	_, resp, err := c.Balance.Get(ctx)
	if err == nil || ErrorDetailsFromContext(ctx) == nil {
		t.Fatal("operation capture missing")
	}
	if ErrorDetailsFromContext(other) != nil {
		t.Fatal("unrelated context modified")
	}
	if resp.Request != unrelated {
		t.Fatal("response request mutated")
	}
}
