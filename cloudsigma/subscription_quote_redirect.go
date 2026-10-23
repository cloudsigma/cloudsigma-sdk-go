package cloudsigma

import "errors"

// ErrSubscriptionQuoteRedirect indicates a refused subscription quote redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrSubscriptionQuoteRedirect = errors.New("subscription quote redirect refused; outcome unknown")
