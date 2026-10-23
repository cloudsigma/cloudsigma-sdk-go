package cloudsigma

import "errors"

// ErrLibraryDriveCloneRedirect indicates a refused library drive clone redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrLibraryDriveCloneRedirect = errors.New("library drive clone redirect refused; outcome unknown")
