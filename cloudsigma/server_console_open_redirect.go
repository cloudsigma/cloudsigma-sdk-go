package cloudsigma

import "errors"

// ErrServerConsoleOpenRedirect indicates a refused console-open redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrServerConsoleOpenRedirect = errors.New("server console open redirect refused; outcome unknown")
