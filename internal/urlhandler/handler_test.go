package urlhandler

import (
	"testing"
	"time"
)

func TestColdBootMagnetArg(t *testing.T) {
	testMagnet := "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=ColdBootTest"
	ch := Init([]string{"torrentclick", testMagnet})

	select {
	case received := <-ch:
		if received != testMagnet {
			t.Errorf("expected %s, got %s", testMagnet, received)
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("timed out waiting for cold boot magnet")
	}
}
