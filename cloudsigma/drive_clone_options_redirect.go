package cloudsigma

import "errors"

// ErrDriveCloneOptionsRedirect indicates a refused presence-aware drive clone redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrDriveCloneOptionsRedirect = errors.New("drive clone options redirect refused; outcome unknown")
