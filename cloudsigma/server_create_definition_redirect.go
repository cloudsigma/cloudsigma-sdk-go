package cloudsigma

import "errors"

// ErrServerCreateDefinitionRedirect indicates a refused server-definition creation redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrServerCreateDefinitionRedirect = errors.New("server definition create redirect refused; outcome unknown")
