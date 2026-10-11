package urlhandler

import (
	"strings"
	"sync"
)

var (
	urlChan  = make(chan string, 10)
	mu       sync.Mutex
	lastURL  string
)

// Init initializes platform URL event handling and inspects startup arguments.
func Init(args []string) <-chan string {
	registerPlatformURLHandler()

	// Inspect os.Args for cold-boot URLs
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if strings.HasPrefix(strings.ToLower(trimmed), "magnet:") {
			dispatchURL(trimmed)
			break
		}
	}

	return urlChan
}

// dispatchURL handles incoming URLs and avoids immediate duplicates.
func dispatchURL(u string) {
	trimmed := strings.TrimSpace(u)
	if trimmed == "" {
		return
	}

	mu.Lock()
	if trimmed == lastURL {
		mu.Unlock()
		return
	}
	lastURL = trimmed
	mu.Unlock()

	select {
	case urlChan <- trimmed:
	default:
		// Queue full, drop or drain
	}
}
