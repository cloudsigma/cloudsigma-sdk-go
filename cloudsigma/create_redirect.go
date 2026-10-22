package cloudsigma

import "errors"

// ErrDriveCreateRedirect indicates a refused drive creation redirect. The initial
// POST outcome is unknown; callers must not blindly retry.
var ErrDriveCreateRedirect = errors.New("drive create redirect refused; outcome unknown")

// ErrServerCreateRedirect indicates a refused server creation redirect. The initial
// POST outcome is unknown; callers must not blindly retry.
var ErrServerCreateRedirect = errors.New("server create redirect refused; outcome unknown")
