package cloudsigma

import "errors"

// ErrDriveCloneRedirect indicates a refused drive clone redirect. The initial
// POST outcome is unknown; callers must not blindly retry.
var ErrDriveCloneRedirect = errors.New("drive clone redirect refused; outcome unknown")

// ErrServerCloneRedirect indicates a refused server clone redirect, including
// CloneWithAvoid. The initial POST outcome is unknown; do not blindly retry.
var ErrServerCloneRedirect = errors.New("server clone redirect refused; outcome unknown")
