package cloudsigma

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// DownloadOptions explicitly bounds a full binary download. Redirects default
// to denied. AllowRedirect is a trusted caller decision, NOT provider approval
// or a DNS/private-address/rebinding defense. It receives copies of URLs.
// Transport is optional trusted injection; nil uses isolated TLS, no proxy,
// no cookies and no automatic compression. Shared client transport is untouched.
type DownloadOptions struct {
	MaxBytes      int64
	Timeout       time.Duration
	MaxRedirects  int
	AllowRedirect func(from, to url.URL) bool
	Transport     http.RoundTripper
}

// Download streams a complete drive image to dst, which remains caller-owned.
// Initial origin is derived only from a standard HTTPS location API endpoint.
// Credentials are sent only to that exact direct origin, including a return hop.
// Every hop gets fresh headers: no cookies or cross-origin authorization.
// SDK policy accepts only 200 application/octet-stream, identity encoding;
// no ranges/resume/206. This is not a claim about all provider status semantics.
// Partial output remains on error. Context is cooperative; hostile transports,
// writers, policy callbacks and Close implementations cannot be bounded here.
func (s *DrivesService) Download(ctx context.Context, id string, dst io.Writer, o DownloadOptions) (int64, error) {
	if ctx == nil || dst == nil || o.MaxBytes <= 0 || o.Timeout <= 0 || o.Timeout > time.Hour || o.MaxRedirects < 0 || o.MaxRedirects > 10 || (o.MaxRedirects > 0 && o.AllowRedirect == nil) {
		return 0, errors.New("invalid download options")
	}
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(id) {
		return 0, errors.New("invalid drive UUID")
	}
	c := s.client
	if c.endpointErr != nil || c.baseURL == nil || c.baseURL.Scheme != "https" || c.baseURL.User != nil || c.baseURL.Path != "/api/2.0/" || c.baseURL.RawQuery != "" || c.baseURL.Fragment != "" || c.baseURL.RawPath != "" || !regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*\.cloudsigma\.com$`).MatchString(c.baseURL.Host) {
		return 0, errors.New("download requires standard location endpoint")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	// Reuse actual credential/request construction without crossing its origin guard.
	template, err := c.NewRequest(http.MethodGet, "drives/"+id+"/download/", nil)
	if err != nil {
		return 0, errors.New("download credentials refused")
	}
	initial := *template.URL
	initial.Host = "direct." + c.baseURL.Host
	rt := o.Transport
	if rt == nil {
		t := &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, DisableCompression: true}
		defer t.CloseIdleConnections()
		rt = t
	}
	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()
	u := initial
	seen := map[string]bool{}
	for hop := 0; ; hop++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if seen[u.String()] {
			return 0, errors.New("download redirect loop")
		}
		seen[u.String()] = true
		q, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return 0, errors.New("download URL refused")
		}
		q.Header.Set("Accept", "application/octet-stream")
		q.Header.Set("Accept-Encoding", "identity")
		q.Header.Set("User-Agent", c.userAgent)
		if u.Scheme == initial.Scheme && u.Host == initial.Host {
			q.Header.Set("Authorization", template.Header.Get("Authorization"))
		}
		r, e := rt.RoundTrip(q)
		if e != nil {
			if r != nil && r.Body != nil {
				_ = r.Body.Close()
			}
			if ctx.Err() != nil {
				return 0, ctx.Err()
			}
			return 0, errors.New("download transport failed")
		}
		if r == nil || r.Body == nil {
			return 0, errors.New("download response missing")
		}
		if r.Request != nil && (r.Request.URL == nil || r.Request.URL.String() != q.URL.String() || r.Request.Host != q.Host) {
			_ = r.Body.Close()
			return 0, errors.New("download final origin refused")
		}
		if r.StatusCode == 301 || r.StatusCode == 302 || r.StatusCode == 303 || r.StatusCode == 307 || r.StatusCode == 308 {
			loc := r.Header.Get("Location")
			_ = r.Body.Close()
			v, e := u.Parse(loc)
			if e != nil || loc == "" || v.Scheme != "https" || v.Host == "" || v.User != nil || v.Fragment != "" || v.RawPath != "" || strings.ContainsAny(v.Host, "\\\r\n") || hop >= o.MaxRedirects || o.AllowRedirect == nil {
				return 0, errors.New("download redirect refused")
			}
			if !o.AllowRedirect(u, *v) {
				return 0, errors.New("download policy refused")
			}
			u = *v
			continue
		}
		n, e := copyDownload(ctx, dst, r, o.MaxBytes)
		_ = r.Body.Close()
		return n, e
	}
}

// DownloadStatusError reports only a status; it never embeds private body data.
type DownloadStatusError struct{ StatusCode int }

func (e *DownloadStatusError) Error() string { return fmt.Sprintf("download status %d", e.StatusCode) }
func copyDownload(ctx context.Context, dst io.Writer, r *http.Response, max int64) (int64, error) {
	if r.StatusCode != 200 {
		return 0, &DownloadStatusError{r.StatusCode}
	}
	typ, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || typ != "application/octet-stream" || (r.Header.Get("Content-Encoding") != "" && r.Header.Get("Content-Encoding") != "identity") || r.Uncompressed {
		return 0, errors.New("download representation refused")
	}
	if r.ContentLength > max {
		return 0, errors.New("download size limit")
	}
	var n int64
	buf := make([]byte, 32*1024)
	empty := 0
	for {
		if err := ctx.Err(); err != nil {
			return n, err
		}
		want := int64(len(buf))
		if max-n < want {
			want = max - n
		}
		if want == 0 {
			want = 1
		}
		k, e := r.Body.Read(buf[:int(want)])
		if err := ctx.Err(); err != nil {
			return n, err
		}
		if int64(k) > max-n {
			return n, errors.New("download size limit")
		}
		if k > 0 {
			empty = 0
			w, we := dst.Write(buf[:k])
			if w < 0 || w > k {
				return n, errors.New("invalid writer count")
			}
			n += int64(w)
			if we != nil {
				return n, we
			}
			if w != k {
				return n, io.ErrShortWrite
			}
		} else {
			empty++
			if empty >= 100 {
				return n, io.ErrNoProgress
			}
		}
		if e != nil {
			if e != io.EOF {
				return n, e
			}
			if r.ContentLength >= 0 && n != r.ContentLength {
				return n, io.ErrUnexpectedEOF
			}
			return n, nil
		}
	}
}
