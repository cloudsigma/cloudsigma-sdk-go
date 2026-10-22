package cloudsigma

import "errors"

// ErrServerPowerRedirect indicates a refused Start, Stop, Shutdown or
// StartWithAvoid redirect. The initial POST outcome is unknown; callers must not
// blindly retry.
var ErrServerPowerRedirect = errors.New("server power redirect refused; outcome unknown")
