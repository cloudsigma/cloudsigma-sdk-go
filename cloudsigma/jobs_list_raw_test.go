package cloudsigma_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type jobsListTransport struct {
	t                   *testing.T
	query, body         string
	code, calls, closed int
	err                 error
	block               bool
}

func (f *jobsListTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.calls++
	if r.Method != http.MethodGet || r.URL.String() != "https://fixture.invalid/custom/api/2.0/jobs/"+f.query {
		f.t.Fatalf("request: %s %s", r.Method, r.URL)
	}
	if r.Body != nil {
		b, e := io.ReadAll(r.Body)
		if e != nil || len(b) != 0 {
			f.t.Fatal("body", string(b), e)
		}
	}
	if r.Header.Get("Authorization") != "Bearer synthetic" || r.Header.Get("Accept") != "application/json" {
		f.t.Fatal("headers", r.Header)
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	if f.err != nil {
		return nil, f.err
	}
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Request-Id", "jobs-fixture")
	rec.Header().Set("Link", `<https://untrusted.invalid/?x=a%26b>; rel="next"`)
	rec.Header().Set("Retry-After", "30")
	rec.WriteHeader(f.code)
	_, _ = rec.WriteString(f.body)
	resp := rec.Result()
	resp.Request = r
	if f.block {
		resp.Body = &jobsBlockedBody{ctx: r.Context(), owner: f}
	} else {
		resp.Body = &jobsListBody{ReadCloser: resp.Body, owner: f}
	}
	return resp, nil
}

type jobsListBody struct {
	io.ReadCloser
	owner *jobsListTransport
}

func (b *jobsListBody) Close() error { b.owner.closed++; return b.ReadCloser.Close() }

type jobsBlockedBody struct {
	ctx   context.Context
	owner *jobsListTransport
}

func (b *jobsBlockedBody) Read([]byte) (int, error) { <-b.ctx.Done(); return 0, b.ctx.Err() }
func (b *jobsBlockedBody) Close() error             { b.owner.closed++; return nil }
func jobsListClient(f *jobsListTransport) *cs.Client {
	return cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithEndpoint("https://fixture.invalid/custom/api/2.0/"), cs.WithHTTPClient(&http.Client{Transport: f}))
}

func TestJobsListRawPublicCollection(t *testing.T) {
	raw, e := os.ReadFile("testdata/jobs_collection_public.json")
	if e != nil {
		t.Fatal(e)
	}
	var want map[string]json.RawMessage
	if e = json.Unmarshal(raw, &want); e != nil {
		t.Fatal(e)
	}
	f := &jobsListTransport{t: t, code: 200, body: string(raw)}
	c := jobsListClient(f)
	// These method values also compile-check the additive API and unchanged Get signature from an external package.
	var list func(context.Context, *cs.ListOptions) (map[string]json.RawMessage, *cs.Response, error) = c.Jobs.ListRaw
	var _ func(context.Context, string) (map[string]json.RawMessage, *cs.Response, error) = c.Jobs.Get
	got, r, e := list(context.Background(), nil)
	if e != nil || r == nil || r.StatusCode != 200 || r.RequestID != "jobs-fixture" || r.Meta != nil || !reflect.DeepEqual(got, want) || f.calls != 1 || f.closed != 1 {
		t.Fatal(got, r, e)
	}
	var objects []map[string]json.RawMessage
	if e = json.Unmarshal(got["objects"], &objects); e != nil || len(objects) != 20 {
		t.Fatal(e, len(objects))
	}
	if !strings.Contains(string(objects[13]["children"]), "b3c8791a-7163-47fd-871d-ce7e74cdea22") || !strings.Contains(string(objects[3]["resources"]), "b9a2e66c-81d1-4de3-aac0-b6f2bde806f1") {
		t.Fatal("populated children/resources missing")
	}
	second, _, e := list(context.Background(), nil)
	if e != nil {
		t.Fatal(e)
	}
	got["objects"][0] = 'X'
	if got["objects"][0] != 'X' || !reflect.DeepEqual(second, want) || f.calls != 2 || f.closed != 2 {
		t.Fatal("ownership")
	}
}
func TestJobsListRawPagination(t *testing.T) {
	for _, tc := range []struct {
		name          string
		opts          *cs.ListOptions
		query         string
		offset, count int
	}{
		{"default", nil, "", 0, 3}, {"all", &cs.ListOptions{}, "?limit=0", 0, 3}, {"first", &cs.ListOptions{Limit: 1}, "?limit=1", 0, 1}, {"middle", &cs.ListOptions{Limit: 1, Offset: 1}, "?limit=1&offset=1", 1, 1}, {"last-valid", &cs.ListOptions{Limit: 1, Offset: 2}, "?limit=1&offset=2", 2, 1}, {"empty", &cs.ListOptions{Limit: 1, Offset: 3}, "?limit=1&offset=3", 3, 0}, {"all-offset", &cs.ListOptions{Offset: 1}, "?limit=0&offset=1", 1, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var before cs.ListOptions
			if tc.opts != nil {
				before = *tc.opts
			}
			objects := []string{}
			for i := 0; i < tc.count; i++ {
				objects = append(objects, fmt.Sprintf(`{"uuid":"job-%d","children":[{"data":{"large":9007199254740993,"decimal":1.2300},"children":[]}],"resources":["/api/2.0/drives/example/"],"unknown":null}`, tc.offset+i))
			}
			body := fmt.Sprintf(`{"meta":{"offset":%d,"total_count":3,"next":"https://untrusted.invalid/?a=1&b=two%%20words","previous":null,"unknown":1e+09},"objects":[%s],"extension":{"flag":false}}`, tc.offset, strings.Join(objects, ","))
			f := &jobsListTransport{t: t, query: tc.query, code: 200, body: body}
			g, r, e := jobsListClient(f).Jobs.ListRaw(context.Background(), tc.opts)
			var want map[string]json.RawMessage
			_ = json.Unmarshal([]byte(body), &want)
			if e != nil || !reflect.DeepEqual(g, want) || r == nil || r.Header.Get("Link") == "" || f.calls != 1 || f.closed != 1 {
				t.Fatal(g, r, e)
			}
			if tc.opts != nil && *tc.opts != before {
				t.Fatal("options mutated")
			}
		})
	}
}
func TestJobsListRawFailures(t *testing.T) {
	for _, body := range []string{"null", "[]", "1", "true", `"text"`, `{"objects":`, ""} {
		t.Run("invalid-"+body, func(t *testing.T) {
			f := &jobsListTransport{t: t, code: 200, body: body}
			g, r, e := jobsListClient(f).Jobs.ListRaw(context.Background(), nil)
			if e == nil || g != nil || f.calls != 1 || f.closed != 1 {
				t.Fatal(g, r, e)
			}
			if body == "null" && r == nil {
				t.Fatal("policy refusal lost response")
			}
			if body != "null" && r != nil {
				t.Fatal("decode return convention changed")
			}
		})
	}
	for _, body := range []string{`{}`, `{"objects":[]}`, `{"objects":null,"meta":null}`} {
		f := &jobsListTransport{t: t, code: 200, body: body}
		g, _, e := jobsListClient(f).Jobs.ListRaw(context.Background(), nil)
		if e != nil || g == nil || f.calls != 1 || f.closed != 1 {
			t.Fatal(g, e)
		}
	}
	for _, code := range []int{401, 403, 404, 429, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			f := &jobsListTransport{t: t, code: code, body: `[{"error_message":"denied","error_point":"jobs","error_type":"synthetic"}]`}
			g, r, e := jobsListClient(f).Jobs.ListRaw(context.Background(), nil)
			var api *cs.ErrorResponse
			if g != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || len(api.Errors) != 1 || api.Errors[0].Message != "denied" || f.calls != 1 || f.closed != 1 || r.Header.Get("Retry-After") != "30" {
				t.Fatal(g, r, e)
			}
		})
	}
	sentinel := errors.New("synthetic transport")
	f := &jobsListTransport{t: t, err: sentinel}
	g, r, e := jobsListClient(f).Jobs.ListRaw(context.Background(), nil)
	if g != nil || r != nil || !errors.Is(e, sentinel) || f.calls != 1 || f.closed != 0 {
		t.Fatal(g, r, e)
	}
	f = &jobsListTransport{t: t}
	c := cs.NewClient(cs.NewTokenCredentialsProvider(""), cs.WithHTTPClient(&http.Client{Transport: f}))
	g, r, e = c.Jobs.ListRaw(context.Background(), nil)
	if g != nil || r != nil || e == nil || f.calls != 0 {
		t.Fatal("credential failure", e)
	}
}
func TestJobsListRawCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &jobsListTransport{t: t}
	g, _, e := jobsListClient(f).Jobs.ListRaw(ctx, nil)
	if g != nil || !errors.Is(e, context.Canceled) || f.calls != 1 {
		t.Fatal(g, e)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	f = &jobsListTransport{t: t, code: 200, block: true}
	g, _, e = jobsListClient(f).Jobs.ListRaw(ctx, nil)
	if g != nil || !errors.Is(e, context.DeadlineExceeded) || f.calls != 1 || f.closed != 1 {
		t.Fatal(g, e)
	}
}
