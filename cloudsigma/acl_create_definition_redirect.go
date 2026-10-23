package cloudsigma

import "errors"

// ErrACLCreateDefinitionRedirect indicates a refused ACL-definition creation redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrACLCreateDefinitionRedirect = errors.New("ACL definition create redirect refused; outcome unknown")
