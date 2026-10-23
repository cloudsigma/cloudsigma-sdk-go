package cloudsigma

import "errors"

// ErrSubscriptionCreateRedirect indicates a refused subscription creation redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrSubscriptionCreateRedirect = errors.New("subscription create redirect refused; outcome unknown")
