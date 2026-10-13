//go:build !darwin

package sysicon

func applyPlatformMode(mode IconMode) {
	// No-op for non-Darwin platforms
}
