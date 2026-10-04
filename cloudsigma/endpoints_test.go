package cloudsigma

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type endpointCredentials struct {
	calls int
	basic bool
}

func (p *endpointCredentials) Retrieve() (Credentials, error) {
	p.calls++
	if p.basic {
		return Credentials{Source: UsernamePasswordCredentialsName, Username: "fixture", Password: "fake"}, nil
	}
	return Credentials{Source: TokenCredentialsName, Token: "fake"}, nil
}

type endpointTransport func(*http.Request) (*http.Response, error)

func (f endpointTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func endpointReply(r *http.Request, location string) *http.Response {
	status := 200
	h := make(http.Header)
	if location != "" {
		status = 302
		h.Set("Location", location)
	}
	return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}
}

func TestEndpointLocations(t *testing.T) {
	for code, host := range map[string]string{"zrh": "zrh.cloudsigma.com", "MNL2": "mnl2.cloudsigma.com", "dub": "ec.servecentric.com", "ruh": "ruh.cld.v2.sa", "adb": "siaflex.cloud", "wdc": "wdc.alpha3cloud.com", "future": "future.cloudsigma.com"} {
		t.Run(code, func(t *testing.T) {
			p := &endpointCredentials{}
			c := NewClient(p, WithLocation(code))
			r, e := c.NewRequest("GET", "profile/", nil)
			if e != nil || r.URL.Host != host || p.calls != 1 {
				t.Fatalf("%v %v", r, e)
			}
		})
	}
	for _, code := range []string{"evil.invalid/x#", "", " x", "x.y", "a/b", "x:443", "x@evil", "-x", "x-", "x?y", "x#"} {
		t.Run("reject/"+code, func(t *testing.T) {
			p := &endpointCredentials{}
			c := NewClient(p, WithLocation(code))
			if _, e := c.NewRequest("GET", "profile/", nil); e == nil || p.calls != 0 {
				t.Fatal("invalid location reached credentials")
			}
		})
	}
}
func TestEndpointCustomValidation(t *testing.T) {
	for _, base := range []string{"https://custom.example/api/2.0/", "https://custom.example:8443/tenant/api/", "https://127.0.0.1:8443/", "https://[::1]:8443/"} {
		t.Run(base, func(t *testing.T) {
			p := &endpointCredentials{}
			c := NewClient(p, WithEndpoint(base))
			if _, e := c.NewRequest("GET", "profile/?limit=1", nil); e != nil {
				t.Fatal(e)
			}
		})
	}
	for _, base := range []string{"http://custom.example/", "//custom.example/", "https:opaque", "https:///api/", "https://user:pass@custom.example/", "https://custom.example/#fragment", "https://custom.example/#", "https://custom.example/?q=x", "https://custom.example/?", "https://custom.example/api", "https://custom.example:/", "https://custom.example:0/", "https://custom.example:65536/", "https://custom.example:0443/", "https://custom.example./", "https://custom_example/", "https://custom.example\\evil/", "https://[garbage]/"} {
		t.Run("reject/"+base, func(t *testing.T) {
			p := &endpointCredentials{}
			c := NewClient(p, WithEndpoint(base))
			if _, e := c.NewRequest("GET", "profile/", nil); e == nil || p.calls != 0 {
				t.Fatal("invalid endpoint reached credentials")
			}
		})
	}
}
func TestEndpointRequestDestinations(t *testing.T) {
	for _, target := range []string{"profile/", "/api/2.0/profile/", "../other/", "https://ZRH.cloudsigma.com:443/api/2.0/profile/"} {
		t.Run(target, func(t *testing.T) {
			c := NewClient(&endpointCredentials{})
			if _, e := c.NewRequest("GET", target, nil); e != nil {
				t.Fatal(e)
			}
		})
	}
	for _, target := range []string{"https://evil.invalid/x", "http://zrh.cloudsigma.com/", "https://zrh.cloudsigma.com:444/", "//zrh.cloudsigma.com/", "https://u@zrh.cloudsigma.com/", "profile/#x", "profile/#", "https:opaque", "https://zrh.cloudsigma.com\\@evil.invalid/"} {
		t.Run("reject/"+target, func(t *testing.T) {
			p := &endpointCredentials{}
			c := NewClient(p)
			if _, e := c.NewRequest("GET", target, nil); e == nil || p.calls != 0 {
				t.Fatal("unsafe destination reached credentials")
			}
		})
	}
}
func TestEndpointDoAndAuth(t *testing.T) {
	for _, basic := range []bool{false, true} {
		p := &endpointCredentials{basic: basic}
		calls := 0
		c := NewClient(p, WithHTTPClient(&http.Client{Transport: endpointTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if basic {
				u, p, ok := r.BasicAuth()
				if !ok || u != "fixture" || p != "fake" {
					t.Fatal("basic changed")
				}
			} else if r.Header.Get("Authorization") != "Bearer fake" {
				t.Fatal("bearer changed")
			}
			return endpointReply(r, ""), nil
		})}))
		r, e := c.NewRequest("GET", "profile/", nil)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = c.Do(context.Background(), r, nil); e != nil || calls != 1 {
			t.Fatal(e)
		}
		r.URL, _ = url.Parse("https://evil.invalid/")
		if _, e = c.Do(context.Background(), r, nil); e == nil || calls != 1 {
			t.Fatal("mutated request escaped")
		}
	}
	p := &endpointCredentials{}
	calls := 0
	c := NewClient(p, WithHTTPClient(&http.Client{Transport: endpointTransport(func(r *http.Request) (*http.Response, error) { calls++; return endpointReply(r, ""), nil })}))
	r, _ := http.NewRequest("GET", "https://zrh.cloudsigma.com/", nil)
	r.Host = "evil.invalid"
	if _, e := c.Do(context.Background(), r, nil); e == nil || calls != 0 {
		t.Fatal("Host override escaped")
	}
}
func TestEndpointRedirects(t *testing.T) {
	for _, tc := range []struct {
		name, target          string
		mutate, stop, allowed bool
	}{
		{"same", "/api/2.0/next/", false, false, true},
		{"same-explicit-port", "https://zrh.cloudsigma.com:443/next/", false, false, true},
		{"host", "https://evil.invalid/", false, false, false},
		{"port", "https://zrh.cloudsigma.com:444/", false, false, false},
		{"downgrade", "http://zrh.cloudsigma.com/", false, false, false},
		{"userinfo", "https://user@zrh.cloudsigma.com/", false, false, false},
		{"callback-mutation", "/next/", true, false, false},
		{"callback-stop", "/next/", false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			callbackCalls := 0
			transport := endpointTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return endpointReply(r, tc.target), nil
				}
				if r.Header.Get("Authorization") != "Bearer fake" {
					t.Fatal("lost same-origin auth")
				}
				return endpointReply(r, ""), nil
			})
			source := &http.Client{Transport: transport, CheckRedirect: func(r *http.Request, via []*http.Request) error {
				callbackCalls++
				if tc.stop {
					return http.ErrUseLastResponse
				}
				if tc.mutate {
					r.URL, _ = url.Parse("https://evil.invalid/")
				}
				return nil
			}}
			c := NewClient(&endpointCredentials{}, WithHTTPClient(source))
			r, _ := c.NewRequest("GET", "profile/", nil)
			_, err := c.Do(context.Background(), r, nil)
			if tc.allowed {
				if err != nil || calls != 2 {
					t.Fatalf("%v calls=%d", err, calls)
				}
			} else if calls != 1 || err == nil {
				t.Fatalf("unexpected redirect result %v calls=%d", err, calls)
			}
			if source.CheckRedirect == nil || source.Transport == nil {
				t.Fatal("caller mutated")
			}
			if strings.HasPrefix(tc.name, "callback") && callbackCalls != 1 {
				t.Fatal("callback ignored")
			}
		})
	}
}

func TestEndpointOptionOrderAndNil(t *testing.T) {
	p := &endpointCredentials{}
	c := NewClient(p, WithLocation("bad/path"), WithEndpoint("https://custom.example/"))
	if _, e := c.NewRequest("GET", "ok", nil); e != nil {
		t.Fatal(e)
	}
	c = NewClient(p, WithEndpoint("http://bad/"), WithLocation("zrh"))
	if _, e := c.NewRequest("GET", "ok", nil); e != nil {
		t.Fatal(e)
	}
	c = NewClient(nil)
	if _, e := c.NewRequest("GET", "ok", nil); e == nil {
		t.Fatal("nil credentials must error")
	}
	if _, e := c.Do(context.Background(), nil, nil); e == nil {
		t.Fatal("nil request must error")
	}
}

func TestEndpointRedirectLimitAndHostMutation(t *testing.T) {
	for _, mutateHost := range []bool{false, true} {
		calls := 0
		source := &http.Client{Transport: endpointTransport(func(r *http.Request) (*http.Response, error) { calls++; return endpointReply(r, "/next/"), nil })}
		if mutateHost {
			source.CheckRedirect = func(r *http.Request, _ []*http.Request) error { r.Host = "evil.invalid"; return nil }
		}
		c := NewClient(&endpointCredentials{}, WithHTTPClient(source))
		r, _ := c.NewRequest("GET", "profile/", nil)
		if _, e := c.Do(context.Background(), r, nil); e == nil {
			t.Fatal("redirect must stop")
		}
		expected := 10
		if mutateHost {
			expected = 1
		}
		if calls != expected {
			t.Fatalf("calls=%d want %d", calls, expected)
		}
		if !mutateHost && source.CheckRedirect != nil {
			t.Fatal("mutated caller callback")
		}
		if _, ok := source.Transport.(endpointTransport); !ok {
			t.Fatal("replaced caller transport")
		}
	}
}

func TestEndpointParserErrorsDoNotEcho(t *testing.T) {
	for _, input := range []string{
		"https://fake-user:fixture-password-marker@host:bad/",
		"https://host/%zz?token=fixture-query-marker",
	} {
		for _, option := range []bool{true, false} {
			p := &endpointCredentials{}
			c := NewClient(p)
			target := input
			if option {
				WithEndpoint(input)(c)
				target = "profile/"
			}
			_, err := c.NewRequest("GET", target, nil)
			if err == nil {
				t.Fatal("malformed URL accepted")
			}
			for _, marker := range []string{input, "fixture-password-marker", "fixture-query-marker", "fake-user"} {
				if strings.Contains(err.Error(), marker) {
					t.Fatal("parser error echoed rejected input")
				}
			}
			if _, ok := err.(*url.Error); ok {
				t.Fatal("raw URL error retained")
			}
			if p.calls != 0 {
				t.Fatal("invalid input retrieved credentials")
			}
			if option && c.baseURL != nil {
				t.Fatal("failed endpoint parser retained URL")
			}
		}
	}
}

func TestEndpointFixtureCancellation(t *testing.T) {
	for _, preCancelled := range []bool{true, false} {
		calls := 0
		requestContext, cancel := context.WithCancel(context.Background())
		transport := fixtureTransport{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; cancel(); _, _ = w.Write([]byte(`{}`)) })}
		c := NewClient(&endpointCredentials{}, WithHTTPClient(&http.Client{Transport: transport}))
		req, err := c.NewRequest("GET", "profile/", nil)
		if err != nil {
			t.Fatal(err)
		}
		if preCancelled {
			cancel()
		}
		resp, err := c.Do(requestContext, req, nil)
		cancel()
		if err != context.Canceled || resp != nil {
			t.Fatalf("expected cancellation, got response=%v error=%v", resp, err)
		}
		expected := 1
		if preCancelled {
			expected = 0
		}
		if calls != expected {
			t.Fatalf("handler calls=%d, want %d", calls, expected)
		}
	}
}
