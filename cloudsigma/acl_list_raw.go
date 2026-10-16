package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ListRaw retrieves one complete ACL collection envelope, including grantees
// omitted by the legacy ACL model. It leaves List and the shared model unchanged.
// Nil options omit pagination parameters; a nonnil zero Limit sends limit=0.
// Raw values retain presence and numeric tokens and are caller-owned. No links
// are followed and Response.Meta is not populated. Null/nonobject responses are
// refused; envelope and field schemas are not validated. Raw preservation does
// not establish permission semantics or authorize readonly fields in writes.
// Docs: https://docs.cloudsigma.com/en/latest/acls.html#listing
func (s *ACLsService) ListRaw(ctx context.Context, opts *ListOptions) (map[string]json.RawMessage, *Response, error) {
	query := url.Values{}
	if opts != nil {
		query.Set("limit", strconv.Itoa(opts.Limit))
		if opts.Offset != 0 {
			query.Set("offset", strconv.Itoa(opts.Offset))
		}
	}
	path := "acls/"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	resp, err := s.client.Do(ctx, req, &out)
	if err != nil {
		return nil, resp, err
	}
	if out == nil {
		return nil, resp, fmt.Errorf("expected ACL collection response object")
	}
	return out, resp, nil
}
