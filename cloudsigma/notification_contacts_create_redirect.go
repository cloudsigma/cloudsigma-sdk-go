package cloudsigma

import "errors"

// ErrNotificationContactsCreateRedirect indicates a refused contact create redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrNotificationContactsCreateRedirect = errors.New("notification contacts create redirect refused; outcome unknown")
