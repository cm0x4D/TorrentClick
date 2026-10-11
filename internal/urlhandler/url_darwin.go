//go:build darwin

package urlhandler

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include "darwin_url.h"
*/
import "C"

//export handleDarwinURL
func handleDarwinURL(urlStr *C.char) {
	if urlStr == nil {
		return
	}
	u := C.GoString(urlStr)
	dispatchURL(u)
}

func registerPlatformURLHandler() {
	C.setupDarwinURLHandler()
}
