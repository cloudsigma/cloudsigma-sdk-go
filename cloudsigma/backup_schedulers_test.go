package cloudsigma

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type schedulerReadWire struct {
	t           *testing.T
	path, body  string
	code, calls int
	err         error
}

func (f *schedulerReadWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	if q.Body != nil {
		data, err := io.ReadAll(q.Body)
		if err != nil || len(data) != 0 {
			f.t.Fatal("nonempty request body", err)
		}
	}
	if q.URL.Fragment != "" {
		f.t.Fatal("body or fragment")
	}
	if q.Method != "GET" || q.URL.RequestURI() != f.path {
		f.t.Fatalf("route %s %s", q.Method, q.URL.RequestURI())
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.body)), Request: q}, nil
}
func readClient(f *schedulerReadWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestBackupSchedulerReadLists(t *testing.T) {
	cases := []struct {
		name        string
		detail      bool
		typ, fields string
	}{
		{"defaultList", false, "", ""}, {"ListBackup", false, "backup", "uuid,name"}, {"defaultListDetail", true, "", ""}, {"ListDetailSnapshot", true, "snapshot", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := "/api/2.0/backupschedulers/"
			if tc.detail {
				path += "detail/"
			}
			var opts *BackupSchedulerListOptions
			if tc.typ != "" {
				opts = &BackupSchedulerListOptions{Type: tc.typ, Fields: tc.fields, ListOptions: ListOptions{Limit: 2, Offset: 3}}
				if tc.fields != "" {
					path += "?fields=uuid%2Cname&limit=2&offset=3&type=backup"
				} else {
					path += "?limit=2&offset=3&type=snapshot"
				}
			}
			f := &schedulerReadWire{t: t, path: path, code: 200, body: `{"meta":{"limit":2,"offset":3,"total_count":9},"objects":[{"uuid":"synthetic","is_enabled":false,"initial_backup":null,"retention_policy":{},"incremental_backup":{"minute":0},"unknown":9007199254740993}]}`}
			c := readClient(f)
			var rows []BackupScheduler
			var r *Response
			var e error
			if tc.detail {
				rows, r, e = c.BackupSchedulers.ListDetail(context.Background(), opts)
			} else {
				rows, r, e = c.BackupSchedulers.List(context.Background(), opts)
			}
			if e != nil || r == nil || r.StatusCode != 200 || r.Meta == nil || len(rows) != 1 || f.calls != 1 {
				t.Fatalf("read %v %v %v", rows, r, e)
			}
			if string(rows[0]["retention_policy"]) != "{}" || string(rows[0]["incremental_backup"]) != `{"minute":0}` || string(rows[0]["is_enabled"]) != "false" || string(rows[0]["initial_backup"]) != "null" || string(rows[0]["unknown"]) != "9007199254740993" {
				t.Fatal("raw values lost")
			}
			if r.Meta.Limit != 2 || r.Meta.Offset != 3 || r.Meta.TotalCount != 9 {
				t.Fatal("meta lost")
			}
		})
	}
}

func TestBackupSchedulerGetAndFailures(t *testing.T) {
	id := "eb352f24-3a72-47a9-96cc-7b508ef52d8e"
	f := &schedulerReadWire{t: t, path: "/api/2.0/backupschedulers/" + id + "/", code: 200, body: `{"uuid":"eb352f24-3a72-47a9-96cc-7b508ef52d8e","is_enabled":true,"incremental_backup":{"minute":0}}`}
	c := readClient(f)
	row, response, e := c.BackupSchedulers.Get(context.Background(), id)
	if e != nil || response == nil || response.StatusCode != 200 || f.calls != 1 || string(row["uuid"]) != `"`+id+`"` || string(row["incremental_backup"]) != `{"minute":0}` || string(row["is_enabled"]) != "true" {
		t.Fatal(e)
	}
	calls := f.calls
	for _, bad := range []string{"", "..", "../../servers/x", "x?do=delete", "%2f", "x\\y", "x\n"} {
		if out, resp, err := c.BackupSchedulers.Get(context.Background(), bad); err == nil || out != nil || resp != nil {
			t.Fatal("unsafe accepted")
		}
	}
	if calls != f.calls {
		t.Fatal("invalid dispatched")
	}
	sentinel := errors.New("synthetic")
	f.err = sentinel
	if _, _, e = c.BackupSchedulers.Get(context.Background(), id); !errors.Is(e, sentinel) {
		t.Fatal(e)
	}
	f.err = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, e = c.BackupSchedulers.Get(ctx, id); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	f.code = 404
	f.body = `[]`
	if row, r, e := c.BackupSchedulers.Get(context.Background(), id); e == nil || row != nil || r == nil || r.StatusCode != 404 {
		t.Fatal("error lost")
	}
	f.code = 200
	f.body = `{`
	if _, _, e = c.BackupSchedulers.Get(context.Background(), id); e == nil {
		t.Fatal("badjson accepted")
	}
}

func TestBackupSchedulerListBoundaries(t *testing.T) {
	for _, detail := range []bool{false, true} {
		name := "List"
		path := "/api/2.0/backupschedulers/"
		if detail {
			name = "ListDetail"
			path += "detail/"
		}
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				name, body    string
				code          int
				fail, nilRows bool
			}{
				{"empty", `{"objects":[]}`, 200, false, false}, {"null", `{"objects":null,"meta":null}`, 200, false, true}, {"missing", `{}`, 200, false, true}, {"nullEnvelope", `null`, 200, false, true}, {"badJSON", `{`, 200, true, true}, {"wrongEnvelope", `[]`, 200, true, true}, {"wrongObjects", `{"objects":{}}`, 200, true, true}, {"httpError", `[]`, 403, true, true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					f := &schedulerReadWire{t: t, path: path, code: tc.code, body: tc.body}
					c := readClient(f)
					method := c.BackupSchedulers.List
					if detail {
						method = c.BackupSchedulers.ListDetail
					}
					rows, r, e := method(context.Background(), nil)
					if (e != nil) != tc.fail || f.calls != 1 {
						t.Fatal("outcome", e)
					}
					if !tc.fail {
						if r == nil || r.Meta != nil || len(rows) != 0 || (rows == nil) != tc.nilRows {
							t.Fatal("permissive shape changed")
						}
					} else if tc.code == 403 {
						if r == nil || r.StatusCode != 403 {
							t.Fatal("HTTP error response lost")
						}
					} else if r != nil {
						t.Fatal("decode error convention")
					}
				})
			}
			for _, cancelled := range []bool{false, true} {
				f := &schedulerReadWire{t: t}
				sentinel := errors.New("transport")
				f.err = sentinel
				ctx := context.Background()
				if cancelled {
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(ctx)
					cancel()
				}
				c := readClient(f)
				method := c.BackupSchedulers.List
				if detail {
					method = c.BackupSchedulers.ListDetail
				}
				_, _, e := method(ctx, nil)
				if cancelled {
					if !errors.Is(e, context.Canceled) {
						t.Fatal(e)
					}
				} else if !errors.Is(e, sentinel) {
					t.Fatal(e)
				}
			}
			opts := &BackupSchedulerListOptions{Type: "backup&x=1/?#%", Fields: "uuid,&extra=x/#", ListOptions: ListOptions{Limit: 2, Offset: 3}}
			q := url.Values{"type": {opts.Type}, "fields": {opts.Fields}, "limit": {"2"}, "offset": {"3"}}
			f := &schedulerReadWire{t: t, path: path + "?" + q.Encode(), code: 200, body: `{"objects":[]}`}
			c := readClient(f)
			method := c.BackupSchedulers.List
			if detail {
				method = c.BackupSchedulers.ListDetail
			}
			if _, _, e := method(context.Background(), opts); e != nil || f.calls != 1 {
				t.Fatal("query encoding", e)
			}
		})
	}
}
