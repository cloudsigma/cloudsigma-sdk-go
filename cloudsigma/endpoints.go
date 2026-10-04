package cloudsigma

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var locationLabel = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// WithEndpoint explicitly selects a trusted HTTPS API base URL. The caller is
// responsible for trusting that server with credentials. Paths must end in '/';
// queries, fragments and userinfo are not endpoint configuration. Explicit ports
// and IP literals are supported. Invalid options fail NewRequest/Do, since the
// existing NewClient signature cannot return an error. Last endpoint option wins.
func WithEndpoint(endpoint string) ClientOption {
	return func(c *Client) {
		u, err := url.Parse(endpoint)
		if err != nil {
			c.baseURL = nil
			c.endpointErr = fmt.Errorf("invalid API endpoint")
			return
		}
		if strings.ContainsAny(endpoint, "#\\") {
			err = fmt.Errorf("ambiguous API endpoint")
		}
		if err == nil {
			err = validateEndpoint(u)
		}
		c.baseURL = u
		c.endpointErr = err
	}
}

func validateURL(u *url.URL) error {
	if u == nil || u.Scheme != "https" || u.Host == "" || u.Opaque != "" || u.User != nil || u.Fragment != "" || u.RawFragment != "" {
		return fmt.Errorf("API destination must be an absolute HTTPS URL without userinfo or fragment")
	}
	host := u.Hostname()
	if host == "" || strings.ContainsAny(u.Host, `\%`) || strings.HasSuffix(u.Host, ":") {
		return fmt.Errorf("invalid API host")
	}

	if strings.HasPrefix(u.Host, "[") {
		if net.ParseIP(host) == nil || !strings.Contains(host, ":") {
			return fmt.Errorf("brackets require an IPv6 literal")
		}
	} else if strings.ContainsAny(u.Host, "[]") || strings.Count(u.Host, ":") > 1 {
		return fmt.Errorf("ambiguous API authority")
	}
	if net.ParseIP(host) == nil {
		for _, label := range strings.Split(host, ".") {
			if !locationLabel.MatchString(strings.ToLower(label)) {
				return fmt.Errorf("invalid API hostname")
			}
		}
		if len(host) > 253 || strings.ContainsAny(u.Host, "[]:") && u.Port() == "" {
			return fmt.Errorf("invalid API hostname")
		}
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != port {
			return fmt.Errorf("invalid API port")
		}
	}
	return nil
}

func validateEndpoint(u *url.URL) error {
	if err := validateURL(u); err != nil {
		return err
	}
	if u.RawQuery != "" || u.ForceQuery || !strings.HasSuffix(u.Path, "/") {
		return fmt.Errorf("API endpoint must have a trailing slash and no query")
	}
	return nil
}

func effectivePort(u *url.URL) string {
	if u.Port() == "" {
		return "443"
	}
	return u.Port()
}
func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Hostname(), b.Hostname()) && effectivePort(a) == effectivePort(b) && a.Scheme == b.Scheme
}

func (c *Client) validateDestination(req *http.Request) error {
	if c.endpointErr != nil {
		return c.endpointErr
	}
	if err := validateEndpoint(c.baseURL); err != nil {
		return err
	}
	if req == nil {
		return fmt.Errorf("nil API request")
	}
	if err := validateURL(req.URL); err != nil {
		return err
	}
	if !sameOrigin(c.baseURL, req.URL) {
		return fmt.Errorf("API request destination differs from configured origin")
	}
	if req.Host != "" && !strings.EqualFold(req.Host, req.URL.Host) {
		return fmt.Errorf("API request Host override differs from URL")
	}
	return nil
}

// guardTransport checks again after a user redirect callback and immediately
// before delegation. User-provided transports remain trusted to honor the URL.
type guardTransport struct {
	client *Client
	next   http.RoundTripper
}

func (t guardTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.client.validateDestination(req); err != nil {
		return nil, err
	}
	return t.next.RoundTrip(req)
}

func (c *Client) guardedHTTPClient() *http.Client {
	source := c.httpClient
	if source == nil {
		source = http.DefaultClient
	}
	copyClient := *source
	next := source.Transport
	if next == nil {
		next = http.DefaultTransport
	}
	copyClient.Transport = guardTransport{c, next}
	copyClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := c.validateDestination(req); err != nil {
			return err
		}
		if source.CheckRedirect != nil {
			if err := source.CheckRedirect(req, via); err != nil {
				return err
			}
		} else if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		return c.validateDestination(req)
	}
	return &copyClient
}
