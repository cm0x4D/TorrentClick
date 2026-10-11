package sysdark

// WatcherChan receives updates when the OS appearance changes.
var WatcherChan = make(chan bool, 5)

// StartWatcher starts background listening for OS theme changes.
func StartWatcher() <-chan bool {
	startPlatformWatcher()
	return WatcherChan
}
