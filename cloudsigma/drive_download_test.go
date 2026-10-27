package cloudsigma

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type downloadWire func(*http.Request) (*http.Response, error)

func (w downloadWire) RoundTrip(q *http.Request) (*http.Response, error) { return w(q) }

type downloadBody struct {
	io.Reader
	closed int
}

func (b *downloadBody) Close() error { b.closed++; return nil }

type downloadWriter struct{}

func (downloadWriter) Write([]byte) (int, error) { return 0, errors.New("writer") }

func TestDownloadStatusError(t *testing.T) {
	err := (&DownloadStatusError{StatusCode: http.StatusBadGateway}).Error()
	if err != "download status 502" {
		t.Fatalf("unexpected error: %q", err)
	}
}

func TestDriveDownload(t *testing.T) {
	for _, mode := range []string{"binary", "default-deny", "cross", "return", "http", "userinfo", "loop", "hops", "size", "truncated", "writer", "cancel", "contenttype", "encoding", "status", "host", "nilurl", "response-error"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancel" {
				cancel()
			}
			n := 0
			bs := []*downloadBody{}
			original := &http.Client{Timeout: 7 * time.Second}
			c := NewClient(NewUsernamePasswordCredentialsProvider("synthetic", "synthetic"), WithHTTPClient(original))
			o := DownloadOptions{MaxBytes: 5, Timeout: time.Second, MaxRedirects: 2, AllowRedirect: func(a, b url.URL) bool { return true }}
			if mode == "default-deny" {
				o.MaxRedirects = 0
				o.AllowRedirect = nil
			}
			o.Transport = downloadWire(func(q *http.Request) (*http.Response, error) {
				n++
				if q.Method != "GET" || q.Body != nil || q.Header.Get("Cookie") != "" || q.Header.Get("Accept-Encoding") != "identity" {
					t.Fatal("request")
				}
				if n == 1 && q.URL.RequestURI() != "/api/2.0/drives/11111111-1111-1111-1111-111111111111/download/" {
					t.Fatal("URI")
				}
				if q.URL.Host == "direct.zrh.cloudsigma.com" {
					u, p, ok := q.BasicAuth()
					if !ok || u != "synthetic" || p != "synthetic" {
						t.Fatal("auth")
					}
				} else if q.Header.Get("Authorization") != "" {
					t.Fatal("foreign auth")
				}
				b := &downloadBody{Reader: strings.NewReader("\x00\xffabc")}
				bs = append(bs, b)
				r := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/octet-stream"}}, Body: b, Request: q, ContentLength: 5}
				switch mode {
				case "default-deny", "cross", "return", "http", "userinfo", "loop", "hops":
					if n == 1 || mode == "hops" || mode == "return" && n == 2 {
						r.StatusCode = 302
						r.Header.Set("Location", "https://blob.example/image")
						if mode == "http" {
							r.Header.Set("Location", "http://blob.example/x")
						}
						if mode == "userinfo" {
							r.Header.Set("Location", "https://u:p@blob.example/x")
						}
						if mode == "loop" {
							r.Header.Set("Location", q.URL.String())
						}
						if mode == "hops" {
							r.Header.Set("Location", q.URL.String()+"x")
						}
						if mode == "return" && n == 2 {
							r.Header.Set("Location", "https://direct.zrh.cloudsigma.com/returned")
						}
					}
				case "size":
					r.ContentLength = -1
					b.Reader = strings.NewReader("123456")
				case "truncated":
					r.ContentLength = 5
					b.Reader = strings.NewReader("123")
				case "contenttype":
					r.Header.Set("Content-Type", "text/html")
				case "encoding":
					r.Header.Set("Content-Encoding", "gzip")
				case "status":
					r.StatusCode = 206
				case "host":
					z := q.Clone(ctx)
					z.Host = "foreign"
					r.Request = z
				case "nilurl":
					r.Request = &http.Request{}
				case "response-error":
					return r, errors.New("private")
				}
				return r, nil
			})
			var out bytes.Buffer
			var dst io.Writer = &out
			if mode == "writer" {
				dst = downloadWriter{}
			}
			_, err := c.Drives.Download(ctx, "11111111-1111-1111-1111-111111111111", dst, o)
			ok := mode == "binary" || mode == "cross" || mode == "return"
			if ok {
				if err != nil || !bytes.Equal(out.Bytes(), []byte{0, 255, 'a', 'b', 'c'}) {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal("refusal")
			}
			for _, b := range bs {
				if b.closed != 1 {
					t.Fatal("closure")
				}
			}
			if mode == "cancel" && n != 0 {
				t.Fatal("dispatch")
			}
			if original.Timeout != 7*time.Second || original.Transport != nil || original.Jar != nil {
				t.Fatal("mutated client")
			}
		})
	}
}
func TestDriveDownloadInvalid(t *testing.T) {
	c := NewClient(NewUsernamePasswordCredentialsProvider("synthetic", "synthetic"))
	for _, o := range []DownloadOptions{{}, {MaxBytes: -1, Timeout: time.Second}, {MaxBytes: 1, Timeout: time.Second, MaxRedirects: 1}, {MaxBytes: 1, Timeout: 2 * time.Hour}} {
		o.Transport = downloadWire(func(*http.Request) (*http.Response, error) { t.Fatal("dispatch"); return nil, nil })
		if _, e := c.Drives.Download(context.Background(), "11111111-1111-1111-1111-111111111111", io.Discard, o); e == nil {
			t.Fatal("options")
		}
	}
}

type downloadCancelReader struct{ cancel context.CancelFunc }

func (r downloadCancelReader) Read(b []byte) (int, error) { r.cancel(); b[0] = 1; return 1, nil }
func TestDriveDownloadMidstreamCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &downloadBody{Reader: downloadCancelReader{cancel}}
	c := NewClient(NewUsernamePasswordCredentialsProvider("synthetic", "synthetic"))
	var dst bytes.Buffer
	_, e := c.Drives.Download(ctx, "11111111-1111-1111-1111-111111111111", &dst, DownloadOptions{MaxBytes: 5, Timeout: time.Second, Transport: downloadWire(func(q *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, ContentLength: -1, Header: http.Header{"Content-Type": []string{"application/octet-stream"}}, Body: b, Request: q}, nil
	})})
	if e != context.Canceled || b.closed != 1 || dst.Len() != 0 {
		t.Fatal(e, b.closed, dst.Len())
	}
}
