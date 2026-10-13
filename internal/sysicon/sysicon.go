package sysicon

// IconMode defines how the application icons are presented on macOS.
type IconMode string

const (
	ModeBoth IconMode = "both" // Both Dock and Menu Bar Tray icons
	ModeTray IconMode = "tray" // Menu Bar Tray icon only (Dock icon hidden)
	ModeDock IconMode = "dock" // Dock icon only (Menu Bar Tray icon hidden)
)

// Normalize returns a valid IconMode from a raw string, defaulting to ModeBoth.
func Normalize(raw string) IconMode {
	switch raw {
	case string(ModeTray):
		return ModeTray
	case string(ModeDock):
		return ModeDock
	default:
		return ModeBoth
	}
}

// ApplyMode configures the visibility of the Dock and System Tray icons.
func ApplyMode(mode IconMode) {
	applyPlatformMode(Normalize(string(mode)))
}
