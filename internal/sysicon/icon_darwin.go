//go:build darwin

package sysicon

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#include "icon_darwin.h"
*/
import "C"

func applyPlatformMode(mode IconMode) {
	showDock := mode == ModeBoth || mode == ModeDock
	showTray := mode == ModeBoth || mode == ModeTray

	var cShowDock, cShowTray C.int
	if showDock {
		cShowDock = 1
	}
	if showTray {
		cShowTray = 1
	}

	C.setDarwinIconDisplay(cShowDock, cShowTray)
}
