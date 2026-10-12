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

func TestColdBootTorrentFileArg(t *testing.T) {
	testTorrent := "/Users/test/Downloads/ubuntu.torrent"
	ch := Init([]string{"torrentclick", testTorrent})

	select {
	case received := <-ch:
		if received != testTorrent {
			t.Errorf("expected %s, got %s", testTorrent, received)
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("timed out waiting for cold boot torrent file")
	}
}

func TestDispatchNormalizeFileURL(t *testing.T) {
	fileURL := "file:///Users/test/Downloads/sample.torrent"
	expectedPath := "/Users/test/Downloads/sample.torrent"
	dispatchURL(fileURL)

	select {
	case received := <-urlChan:
		if received != expectedPath {
			t.Errorf("expected normalized path %s, got %s", expectedPath, received)
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("timed out waiting for dispatched file URL")
	}
}
