package cloudsigma

import "errors"

// ErrGlobalContextUpdateRedirect indicates a refused global context update redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrGlobalContextUpdateRedirect = errors.New("global context update redirect refused; outcome unknown")
