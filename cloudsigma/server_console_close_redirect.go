package cloudsigma

import "errors"

// ErrServerConsoleCloseRedirect indicates a refused console-close redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrServerConsoleCloseRedirect = errors.New("server console close redirect refused; outcome unknown")
