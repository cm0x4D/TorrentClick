package magnet

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestParseTorrentBytes(t *testing.T) {
	// "http://tracker.example.com/" is 27 bytes
	// "udp://tracker2.example.com/" is 27 bytes
	rawTorrent := "d8:announce27:http://tracker.example.com/13:announce-listll27:http://tracker.example.com/el27:udp://tracker2.example.com/ee4:infod6:lengthi1048576e4:name12:testfile.iso12:piece lengthi16384e6:pieces20:12345678901234567890ee"

	m, err := ParseTorrentBytes([]byte(rawTorrent), "/tmp/testfile.torrent")
	if err != nil {
		t.Fatalf("unexpected error parsing torrent bytes: %v", err)
	}

	rawInfo := "d6:lengthi1048576e4:name12:testfile.iso12:piece lengthi16384e6:pieces20:12345678901234567890e"
	expectedSum := sha1.Sum([]byte(rawInfo))
	expectedHash := hex.EncodeToString(expectedSum[:])

	if m.InfoHash != expectedHash {
		t.Errorf("expected info hash %s, got %s", expectedHash, m.InfoHash)
	}

	if m.DisplayName != "testfile.iso" {
		t.Errorf("expected display name 'testfile.iso', got %s", m.DisplayName)
	}

	if m.Size != "1048576" {
		t.Errorf("expected size 1048576, got %s", m.Size)
	}

	if len(m.Trackers) != 2 {
		t.Fatalf("expected 2 trackers, got %d", len(m.Trackers))
	}

	if m.Trackers[0] != "http://tracker.example.com/" {
		t.Errorf("expected tracker 0 'http://tracker.example.com/', got %s", m.Trackers[0])
	}
	if m.Trackers[1] != "udp://tracker2.example.com/" {
		t.Errorf("expected tracker 1 'udp://tracker2.example.com/', got %s", m.Trackers[1])
	}

	if m.TorrentPath != "/tmp/testfile.torrent" {
		t.Errorf("expected TorrentPath '/tmp/testfile.torrent', got %s", m.TorrentPath)
	}

	expectedPrefix := "magnet:?xt=urn:btih:" + expectedHash
	if len(m.Raw) < len(expectedPrefix) || m.Raw[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected synthesized magnet to start with %s, got %s", expectedPrefix, m.Raw)
	}
}

func TestParseTorrentFileAndParseDispatch(t *testing.T) {
	tmpDir := t.TempDir()
	torrentPath := filepath.Join(tmpDir, "sample.torrent")

	// "http://test.tracker.org" is 23 bytes
	// "ubuntu.iso" is 10 bytes
	rawTorrent := "d8:announce23:http://test.tracker.org4:infod4:name10:ubuntu.iso6:lengthi2048e12:piece lengthi512e6:pieces20:abcdefghij0123456789ee"
	if err := os.WriteFile(torrentPath, []byte(rawTorrent), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Test ParseTorrentFile directly
	m1, err := ParseTorrentFile(torrentPath)
	if err != nil {
		t.Fatalf("ParseTorrentFile error: %v", err)
	}
	if m1.DisplayName != "ubuntu.iso" {
		t.Errorf("expected ubuntu.iso, got %s", m1.DisplayName)
	}

	// 2. Test universal Parse() with file path
	m2, err := Parse(torrentPath)
	if err != nil {
		t.Fatalf("Parse(path) error: %v", err)
	}
	if m2.InfoHash != m1.InfoHash {
		t.Errorf("info hash mismatch between Parse and ParseTorrentFile")
	}

	// 3. Test universal Parse() with file:// URL
	m3, err := Parse("file://" + torrentPath)
	if err != nil {
		t.Fatalf("Parse(file://...) error: %v", err)
	}
	if m3.InfoHash != m1.InfoHash {
		t.Errorf("info hash mismatch for file:// URL")
	}

	// 4. Test template context
	ctx := m2.ToTemplateContext(map[string]string{"cat": "iso"}, "sess-123")
	if ctx["TorrentPath"] != torrentPath {
		t.Errorf("expected context TorrentPath %s, got %v", torrentPath, ctx["TorrentPath"])
	}
	if ctx["TorrentBase64"] == "" {
		t.Errorf("expected non-empty TorrentBase64 in context")
	}
}
