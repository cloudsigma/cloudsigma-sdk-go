package cloudsigma

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// StartWithAvoid adds ordered, best-effort compute-host placement hints using
// server IDs. Nil/empty omit the hint as SDK policy and match Start's wire form.
// No placement guarantee, polling or retries. Duplicates are preserved without
// claiming server duplicate semantics. Caller input must not mutate concurrently.
// Docs: https://docs.cloudsigma.com/en/latest/availability_groups.html#servers-avoid
func (s *ServersService) StartWithAvoid(ctx context.Context, id string, avoid []string) (*ServerAction, *Response, error) {
	return s.doActionAvoid(ctx, id, "start", avoid)
}

// CloneWithAvoid adds ordered, best-effort cloned-drive storage placement hints
// using server or drive IDs (server IDs refer to attached drives). It does not
// select compute hosts. Nil/empty omit the hint as SDK policy and match Clone.
// Payload/decoder limitations of Clone apply. No completion/placement guarantee,
// polling or retries. Duplicates preserved; concurrent caller mutation unsupported.
// Docs: https://docs.cloudsigma.com/en/latest/availability_groups.html#drives-avoid
func (s *ServersService) CloneWithAvoid(ctx context.Context, id string, options ServerCloneRequest, avoid []string) (*Server, *Response, error) {
	return s.cloneAvoid(ctx, id, options, avoid)
}

func serverAvoidQuery(ids []string) (string, error) {
	if len(ids) == 0 {
		return "", nil
	}
	for _, id := range ids {
		if id == "" {
			return "", ErrEmptyArgument
		}
		for _, c := range id {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
				return "", fmt.Errorf("invalid avoid identifier")
			}
		}
	}
	// Encode the documented comma-separated value; no braces or repeated keys.
	return "&avoid=" + url.QueryEscape(strings.Join(ids, ",")), nil
}
