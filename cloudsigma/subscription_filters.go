package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// SubscriptionListOptions selects documented status/resource filters only.
// Empty Status and nil/empty Resources omit their parameters (SDK policy),
// not an assertion that explicit empty parameters have server meaning.
type SubscriptionListOptions struct {
	Status    string
	Resources []string
}

// ListWithOptions reads one subscriptions response without automatic traversal.
// Allowed values follow the public subscriptions listing contract. Invalid
// nonempty values fail before dispatch. Options are never modified or retained.
// Existing List and response projection/mutation behavior remain unchanged.
func (s *SubscriptionsService) ListWithOptions(ctx context.Context, opts *SubscriptionListOptions) ([]Subscription, *Response, error) {
	path, err := subscriptionListPath(opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(subscriptionsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}
	return root.Subscriptions, resp, nil
}

func subscriptionListPath(opts *SubscriptionListOptions) (string, error) {
	query := url.Values{}
	if opts != nil {
		if opts.Status != "" {
			switch opts.Status {
			case "active", "inactive", "expired", "all", "notexpired":
			default:
				return "", fmt.Errorf("unsupported subscription status")
			}
			query.Set("status", opts.Status)
		}
		for _, resource := range opts.Resources {
			switch resource {
			case "dssd", "cpu", "mem", "tx", "ip", "vlan":
			default:
				return "", fmt.Errorf("unsupported subscription resource")
			}
		}
		if len(opts.Resources) > 0 {
			query.Set("resource", strings.Join(opts.Resources, ","))
		}
	}
	path := "subscriptions/"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return path, nil
}
