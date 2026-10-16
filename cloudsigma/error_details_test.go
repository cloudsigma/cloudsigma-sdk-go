package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type detailBody struct {
	io.Reader
	closes int
}

func (b *detailBody) Close() error { b.closes++; return nil }

type detailBroken struct{}

func (detailBroken) Read(p []byte) (int, error) { copy(p, "partial"); return 7, io.ErrUnexpectedEOF }

func TestErrorDetailsActualClient(t *testing.T) {
	for _, body := range []string{`[{"error_message":"bad","error_type":"validation","error_point":null}]`, `[{"error_point":""}]`, `[{"error_point":"name"}]`, `[{"error_point":12}]`, `{"error_point":null}`, `not json`, ``, `[]`} {
		t.Run(body, func(t *testing.T) {
			var legacy, captured error
			for _, opt := range []bool{false, true} {
				b := &detailBody{Reader: strings.NewReader(body)}
				calls := 0
				c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
					calls++
					if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/balance/" {
						t.Fatal("request")
					}
					return &http.Response{StatusCode: 400, Request: q, Body: b, Header: make(http.Header)}, nil
				})}))
				ctx := context.Background()
				if opt {
					ctx = WithErrorDetails(ctx)
				}
				_, r, e := c.Balance.Get(ctx)
				if calls != 1 || b.closes != 1 || r == nil || e == nil {
					t.Fatal("lifecycle", e)
				}
				if !opt {
					legacy = e
					if ErrorDetailsFromContext(ctx) != nil {
						t.Fatal("not opted in")
					}
					continue
				}
				captured = e
				d := ErrorDetailsFromContext(ctx)
				if d == nil || string(d.RawBody()) != body || d.ReadError() != nil {
					t.Fatal("capture")
				}
				var a, typedCaptured *ErrorResponse
				if errors.As(legacy, &a) != errors.As(captured, &typedCaptured) {
					t.Fatal("typed error changed")
				}
				if legacy.Error() != captured.Error() {
					t.Fatal("error string changed")
				}
				if strings.HasPrefix(body, `[{"error_point":`) && !strings.Contains(body, "12") {
					ds, err := d.Array()
					if err != nil {
						t.Fatal(err)
					}
					p, present := ds[0].Point()
					if !present || p == nil {
						t.Fatal("point")
					}
				}
				if strings.Contains(body, `"error_point":null`) {
					var v ErrorDetail
					var err error
					if body[0] == '[' {
						vs, e := d.Array()
						err = e
						if e == nil {
							v = vs[0]
						}
					} else {
						v, err = d.Object()
					}
					p, ok := v.Point()
					if err != nil || !ok || p != nil {
						t.Fatal("null")
					}
				}
			}
		})
	}
}

func TestErrorDetailsDecodeStrictAndOwnership(t *testing.T) {
	for _, s := range []string{`{"error_point":false}`, `{"error_point":1}`, `{"error_point":[]}`, `{"error_point":{}}`, `null`, `[]`, ``, `{"error_point":null} {}`} {
		if _, err := DecodeErrorDetails([]byte(s)).Object(); err == nil {
			t.Fatal("accepted", s)
		}
	}
	for _, s := range []string{`null`, `{}`, `[null]`, `[1]`, `[{"error_point":false}]`, `[] []`} {
		if _, e := DecodeErrorDetails([]byte(s)).Array(); e == nil {
			t.Fatal("array", s)
		}
	}
	for _, s := range []string{`{}`, `{"error_point":null}`, `{"error_point":""}`, `{"error_point":"x"}`} {
		input := []byte(s)
		d := DecodeErrorDetails(input)
		input[0] = '!'
		if string(input) == s || string(d.RawBody()) != s {
			t.Fatal("input alias")
		}
		raw := d.RawBody()
		raw[0] = '?'
		if raw[0] != '?' || string(d.RawBody()) != s {
			t.Fatal("output alias")
		}
		v, e := d.Object()
		if e != nil {
			t.Fatal(e)
		}
		p, ok := v.Point()
		if ok != (s != "{}") {
			t.Fatal("presence")
		}
		if p != nil {
			old := *p
			*p = "changed"
			q, _ := v.Point()
			if *p != "changed" || *q != old {
				t.Fatal("point alias")
			}
		}
	}
}

func TestErrorDetailsReadFailureAndReset(t *testing.T) {
	ctx := WithErrorDetails(context.Background())
	status := 400
	broken := true
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
		var reader io.Reader = strings.NewReader(`{"balance":"1"}`)
		if broken {
			reader = detailBroken{}
		}
		return &http.Response{StatusCode: status, Request: q, Body: &detailBody{Reader: reader}, Header: make(http.Header)}, nil
	})}))
	_, _, e := c.Balance.Get(ctx)
	var typed *ErrorResponse
	if !errors.As(e, &typed) {
		t.Fatal(e)
	}
	d := ErrorDetailsFromContext(ctx)
	if d == nil || string(d.RawBody()) != "partial" || !errors.Is(d.ReadError(), io.ErrUnexpectedEOF) {
		t.Fatal("partial")
	}
	broken = false
	status = 200
	_, _, e = c.Balance.Get(ctx)
	if e != nil || ErrorDetailsFromContext(ctx) != nil {
		t.Fatal("reset", e)
	}
	sentinel := errors.New("transport")
	c = NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(*http.Request) (*http.Response, error) { return nil, sentinel })}))
	_, _, e = c.Balance.Get(ctx)
	if !errors.Is(e, sentinel) || ErrorDetailsFromContext(ctx) != nil {
		t.Fatal("transport")
	}
}

func TestLegacyPointLossCharacterization(t *testing.T) {
	var a, b Error
	if e := json.Unmarshal([]byte(`{"error_point":null}`), &a); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal([]byte(`{"error_point":""}`), &b); e != nil {
		t.Fatal(e)
	}
	if a.Point != b.Point {
		t.Fatal("legacy unexpectedly changed")
	}
}

func TestErrorDetailsCancellationAndNilRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(WithErrorDetails(context.Background()))
	cancel()
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { return nil, q.Context().Err() })}))
	_, _, err := c.Balance.Get(ctx)
	if !errors.Is(err, context.Canceled) || ErrorDetailsFromContext(ctx) != nil {
		t.Fatal("cancel", err)
	}
	if _, err := c.Do(ctx, nil, nil); err == nil {
		t.Fatal("nil request")
	}
}

func TestErrorDetailsPublicVariantsOracle(t *testing.T) {
	for _, token := range []string{`null`, `""`, `"resource"`} {
		body := []byte(`[{"error_message":"This field is required","error_point":` + token + `,"error_type":"validation"}]`)
		details, err := DecodeErrorDetails(body).Array()
		if err != nil || len(details) != 1 {
			t.Fatal(err)
		}
		point, present := details[0].Point()
		if !present || details[0].Message != "This field is required" || details[0].Type != "validation" {
			t.Fatal("public fields")
		}
		got, err := json.Marshal(point)
		if err != nil || string(got) != token {
			t.Fatal("nullable oracle", string(got), token)
		}
	}
}
