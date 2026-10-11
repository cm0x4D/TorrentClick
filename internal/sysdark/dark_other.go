//go:build !darwin

package sysdark

// IsDarkMode returns false on non-Darwin platforms by default.
func IsDarkMode() bool {
	return false
}

func startPlatformWatcher() {
	// No-op
}
