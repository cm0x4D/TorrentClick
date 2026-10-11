//go:build !darwin

package urlhandler

func registerPlatformURLHandler() {
	// Non-darwin platforms receive URLs primarily via os.Args
}
