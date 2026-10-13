package cloudsigma

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type availabilityWire func(*http.Request) (*http.Response, error)

func (f availabilityWire) RoundTrip(q *http.Request) (*http.Response, error) { return f(q) }
func TestAvailabilityGroups(t *testing.T) {
	const id = "313e73a4-592f-48cf-81c4-a6c079d005a5"
	for _, kind := range []string{"servers", "drives"} {
		for _, single := range []bool{false, true} {
			for _, body := range []string{`[]`, `["` + id + `"]`, `[["` + id + `"]]`, `null`, `[null]`, `[[null]]`, `{}`, `[1]`, `["bad"]`, `[`, ``} {
				calls := 0
				c := NewClient(NewTokenCredentialsProvider("synthetic"), WithLocation("next"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
					calls++
					want := "/api/2.0/" + kind + "/availability_groups/"
					if single {
						want += id + "/"
					}
					if q.Method != "GET" || q.URL.RequestURI() != want || q.URL.RawQuery != "" || q.Body != nil && q.Body != http.NoBody {
						t.Fatal("wire")
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
				})}))
				var err error
				var size int
				if single {
					var a []string
					if kind == "servers" {
						a, _, err = c.Servers.GetAvailabilityGroup(context.Background(), id)
					} else {
						a, _, err = c.Drives.GetAvailabilityGroup(context.Background(), id)
					}
					size = len(a)
				} else {
					var a [][]string
					if kind == "servers" {
						a, _, err = c.Servers.ListAvailabilityGroups(context.Background())
					} else {
						a, _, err = c.Drives.ListAvailabilityGroups(context.Background())
					}
					size = len(a)
				}
				valid := body == `[]` || single && body == `["`+id+`"]` || !single && body == `[["`+id+`"]]`
				if (err == nil) != valid {
					t.Fatalf("%s single%v body%q err%v", kind, single, body, err)
				}
				if valid && body != `[]` && size != 1 {
					t.Fatal("shape")
				}
				if calls != 1 {
					t.Fatal("count")
				}
			}
		}
	}
}
func TestAvailabilityErrors(t *testing.T) {
	for _, code := range []int{403, 404, 500, 0} {
		calls := 0
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			if code == 0 {
				return nil, errors.New("synthetic transport")
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[]`)), Request: q}, nil
		})}))
		_, r, e := c.Servers.ListAvailabilityGroups(context.Background())
		if e == nil || calls != 1 || code != 0 && (r == nil || r.StatusCode != code) {
			t.Fatal("error propagation")
		}
	}
	for _, id := range []string{"", "../x", "x?y", "x/y", "not-uuid"} {
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(*http.Request) (*http.Response, error) { t.Fatal("invalid dispatch"); return nil, nil })}))
		if _, _, e := c.Servers.GetAvailabilityGroup(context.Background(), id); e == nil {
			t.Fatal("invalid accepted")
		}
		if _, _, e := c.Drives.GetAvailabilityGroup(context.Background(), id); e == nil {
			t.Fatal("invalid accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { return nil, q.Context().Err() })}))
	if _, _, e := c.Drives.ListAvailabilityGroups(ctx); !errors.Is(e, context.Canceled) {
		t.Fatal("cancel", e)
	}
}
