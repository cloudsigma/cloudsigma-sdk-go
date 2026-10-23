package cloudsigma

import "errors"

// ErrKeypairCreateRedirect indicates a refused keypair-create redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrKeypairCreateRedirect = errors.New("keypair create redirect refused; outcome unknown")
