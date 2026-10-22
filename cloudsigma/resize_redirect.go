package cloudsigma

import "errors"

// ErrDriveResizeRedirect indicates a refused drive resize redirect. The initial
// POST outcome is unknown; callers must not blindly retry.
var ErrDriveResizeRedirect = errors.New("drive resize redirect refused; outcome unknown")
