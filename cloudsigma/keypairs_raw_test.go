package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

type keypairWire func(*http.Request) (*http.Response, error)

func (f keypairWire) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func keypairClient(f keypairWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func keypairReply(q *http.Request, code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}
}

// The same response-preservation oracle runs against legacy methods on baseline
// and opt-in methods on candidate. Legacy failure is projection loss, not a
// claim that a new method compiled on the baseline.
func keypairRead(c *Client, collection bool, opts *ListOptions) (map[string]json.RawMessage, *Response, error) {
	if os.Getenv("KEYPAIR_BASELINE") == "1" {
		var value interface{}
		var r *Response
		var e error
		if collection {
			value, r, e = c.Keypairs.List(context.Background())
		} else {
			value, r, e = c.Keypairs.Get(context.Background(), "key-1")
		}
		if e != nil {
			return nil, r, e
		}
		data, e := json.Marshal(value)
		if e != nil {
			return nil, r, e
		}
		if collection {
			data = append(append([]byte(`{"objects":`), data...), '}')
		}
		var out map[string]json.RawMessage
		e = json.Unmarshal(data, &out)
		return out, r, e
	}
	var value *KeypairRaw
	var r *Response
	var e error
	if collection {
		value, r, e = c.Keypairs.ListRaw(context.Background(), opts)
	} else {
		value, r, e = c.Keypairs.GetRaw(context.Background(), "key-1")
	}
	if e != nil {
		return nil, r, e
	}
	return map[string]json.RawMessage(*value), r, nil
}

func TestKeypairRawProjection(t *testing.T) {
	for _, collection := range []bool{false, true} {
		name := "singleton"
		if collection {
			name = "collection"
		}
		t.Run(name, func(t *testing.T) {
			fixture, e := os.ReadFile("testdata/keypair_" + name + ".json")
			if e != nil {
				t.Fatal(e)
			}
			nested := `{"uuid":"key-1","grantees":[{"uuid":"grantee-1","future":{"n":9007199254740993}}],"permissions":[{"action":"read","scope":["one","two"]}],"tags":[{"uuid":"tag-1","meta":{"nested":null}}],"private_key":null,"future":1.23000000000000000001}`
			if collection {
				nested = `{"objects":[` + nested + `],"meta":{"limit":0,"offset":0,"total_count":1,"next":null},"extension":{"n":9007199254740993}}`
			}
			for _, body := range []string{string(fixture), nested, `{}`, `{"objects":null,"meta":null}`, `{"objects":[],"meta":{}}`} {
				calls := 0
				c := keypairClient(func(q *http.Request) (*http.Response, error) {
					calls++
					want := "/api/2.0/keypairs/key-1/"
					if collection {
						want = "/api/2.0/keypairs/"
					}
					if q.Method != "GET" || q.URL.RequestURI() != want || q.Body != nil && q.Body != http.NoBody {
						t.Fatal("wire", q)
					}
					return keypairReply(q, 200, body), nil
				})
				got, r, e := keypairRead(c, collection, nil)
				if e != nil || r == nil || calls != 1 {
					t.Fatal(e, calls)
				}
				var want map[string]json.RawMessage
				if e = json.Unmarshal([]byte(body), &want); e != nil {
					t.Fatal(e)
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("response projection lost fields: got %s want %s", mustKeypairJSON(got), body)
				}
				// Prove mutation before taking an independent fresh response.
				got["injected"] = json.RawMessage(`true`)
				for k, v := range got {
					if k != "injected" && len(v) > 0 {
						v[0] = 'X'
					}
				}
				if string(got["injected"]) != "true" {
					t.Fatal("mutation did not occur")
				}
				fresh, _, e := keypairRead(c, collection, nil)
				if e != nil || calls != 2 || !reflect.DeepEqual(fresh, want) {
					t.Error("fresh read fidelity/ownership", e)
				}
			}
		})
	}
}
func mustKeypairJSON(v interface{}) string { b, _ := json.Marshal(v); return string(b) }

func TestKeypairRawPagination(t *testing.T) {
	for _, tc := range []struct {
		opts  *ListOptions
		query string
	}{{nil, ""}, {&ListOptions{}, "limit=0"}, {&ListOptions{Limit: 2, Offset: 3}, "limit=2&offset=3"}} {
		calls := 0
		c := keypairClient(func(q *http.Request) (*http.Response, error) {
			calls++
			if q.URL.RawQuery != tc.query || q.URL.Path != "/api/2.0/keypairs/" {
				t.Fatal(q.URL)
			}
			return keypairReply(q, 200, `{"objects":[],"meta":{"limit":0,"next":null}}`), nil
		})
		got, r, e := c.Keypairs.ListRaw(context.Background(), tc.opts)
		if e != nil || calls != 1 || r.Meta != nil || string((*got)["meta"]) != `{"limit":0,"next":null}` {
			t.Fatal(got, r, e)
		}
	}
	c := keypairClient(func(q *http.Request) (*http.Response, error) { t.Fatal("unexpected dispatch"); return nil, nil })
	for _, opts := range []*ListOptions{{Limit: -1}, {Offset: -1}} {
		if _, _, e := c.Keypairs.ListRaw(context.Background(), opts); e == nil {
			t.Fatal("negative accepted")
		}
	}
	for _, id := range []string{"", "..", "a/b", "a?x", "a#x", "a%2fb", "a\n", "é"} {
		if _, _, e := c.Keypairs.GetRaw(context.Background(), id); !errors.Is(e, ErrEmptyArgument) {
			t.Fatal(id, e)
		}
	}
}

func TestKeypairRawErrors(t *testing.T) {
	for _, collection := range []bool{false, true} {
		for _, code := range []int{401, 403, 404, 500} {
			calls := 0
			c := keypairClient(func(q *http.Request) (*http.Response, error) {
				calls++
				return keypairReply(q, code, `[{"error_message":"synthetic","error_point":"keypairs","error_type":"denied"}]`), nil
			})
			got, r, e := keypairRead(c, collection, nil)
			var typed *ErrorResponse
			if got != nil || calls != 1 || !errors.As(e, &typed) || typed.Response != r || r.StatusCode != code || len(typed.Errors) != 1 || typed.Errors[0].Message != "synthetic" || typed.Errors[0].Point != "keypairs" || typed.Errors[0].Type != "denied" {
				t.Fatal(got, r, e)
			}
		}
		for _, body := range []string{"", `null`, `[]`, `42`, `"x"`, `{`} {
			c := keypairClient(func(q *http.Request) (*http.Response, error) { return keypairReply(q, 200, body), nil })
			got, r, e := keypairRead(c, collection, nil)
			if got != nil || r != nil || e == nil {
				t.Fatal("malformed", body, got, r, e)
			}
		}
		sentinel := errors.New("synthetic transport")
		c := keypairClient(func(q *http.Request) (*http.Response, error) { return nil, sentinel })
		if _, _, e := keypairRead(c, collection, nil); !errors.Is(e, sentinel) {
			t.Fatal(e)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		c = keypairClient(func(q *http.Request) (*http.Response, error) { return nil, q.Context().Err() })
		var e error
		if collection {
			_, _, e = c.Keypairs.ListRaw(ctx, nil)
		} else {
			_, _, e = c.Keypairs.GetRaw(ctx, "key-1")
		}
		if !errors.Is(e, context.Canceled) {
			t.Fatal(e)
		}
	}
}
