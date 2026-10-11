//go:build darwin

package sysdark

/*
#cgo LDFLAGS: -framework Cocoa
#include "dark_darwin.h"
*/
import "C"
import "runtime"

//export onDarwinThemeChanged
func onDarwinThemeChanged() {
	select {
	case WatcherChan <- IsDarkMode():
	default:
	}
}

// IsDarkMode returns true if macOS is currently set to dark appearance.
func IsDarkMode() bool {
	return C.isDarwinDarkMode() == 1
}

func startPlatformWatcher() {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		C.startDarwinThemeObserver()
	}()
}
