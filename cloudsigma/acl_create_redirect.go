package cloudsigma

import "errors"

// ErrACLCreateRedirect indicates a refused legacy ACL creation redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrACLCreateRedirect = errors.New("ACL create redirect refused; outcome unknown")
