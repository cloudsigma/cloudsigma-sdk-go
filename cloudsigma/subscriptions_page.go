package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ListRawPage reads one subscription collection page, preserving its complete
// raw envelope. Filters use the same validation as ListRaw. Nil page options
// omit pagination; a nonnil zero value requests limit=0 (all resources).
// Negative pagination is rejected by the SDK. No automatic traversal or
// billing interpretation is performed. Existing listing methods are unchanged.
// Docs: https://docs.cloudsigma.com/en/latest/pagination.html
func (s *SubscriptionsService) ListRawPage(ctx context.Context, filters *SubscriptionListOptions, page *ListOptions) (*SubscriptionsRaw, *Response, error) {
	if page != nil && (page.Limit < 0 || page.Offset < 0) {
		return nil, nil, fmt.Errorf("negative subscription pagination")
	}
	path, err := subscriptionListPath(filters)
	if err != nil {
		return nil, nil, err
	}
	if page != nil {
		u, err := url.Parse(path)
		if err != nil {
			return nil, nil, err
		}
		query := u.Query()
		query.Set("limit", strconv.Itoa(page.Limit))
		if page.Offset != 0 {
			query.Set("offset", strconv.Itoa(page.Offset))
		}
		u.RawQuery = query.Encode()
		path = u.String()
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(SubscriptionsRaw)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
