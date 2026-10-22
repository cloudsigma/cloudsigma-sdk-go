package cloudsigma

import "errors"

// ErrKeypairGenerateRedirect indicates a refused keypair generation redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrKeypairGenerateRedirect = errors.New("keypair generate redirect refused; outcome unknown")
