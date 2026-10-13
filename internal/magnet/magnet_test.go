package magnet

import (
	"encoding/base64"
	"testing"
)

func TestParseValidMagnet(t *testing.T) {
	raw := "magnet:?xt=urn:btih:c12fe1c06bba254a9dc9f519b335380dc1f74c26&dn=Ubuntu+22.04+LTS&tr=http%3A%2F%2Ftracker.example.com%2Fannounce&tr=udp%3A%2F%2Ftracker2.example.com%3A80"
	m, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.InfoHash != "c12fe1c06bba254a9dc9f519b335380dc1f74c26" {
		t.Errorf("expected hash c12fe1c06bba254a9dc9f519b335380dc1f74c26, got %s", m.InfoHash)
	}
	if m.DisplayName != "Ubuntu 22.04 LTS" {
		t.Errorf("expected display name 'Ubuntu 22.04 LTS', got %s", m.DisplayName)
	}
	if len(m.Trackers) != 2 {
		t.Fatalf("expected 2 trackers, got %d", len(m.Trackers))
	}
	if m.Trackers[0] != "http://tracker.example.com/announce" {
		t.Errorf("expected tracker 0, got %s", m.Trackers[0])
	}
}

func TestParseInvalid(t *testing.T) {
	_, err := Parse("http://example.com/file.torrent")
	if err == nil {
		t.Errorf("expected error for non-magnet scheme")
	}

	_, err = Parse("")
	if err == nil {
		t.Errorf("expected error for empty string")
	}
}

func TestParseTorrentScheme(t *testing.T) {
	// A valid minimal torrent payload
	rawTorrent := "d8:announce27:http://tracker.example.com/4:infod6:lengthi1048576e4:name12:testfile.iso12:piece lengthi16384e6:pieces20:12345678901234567890ee"
	b64Std := base64.StdEncoding.EncodeToString([]byte(rawTorrent))

	// 1. Test torrentclick://add?name=...&data=... (standard base64 with URL encoding)
	u1 := "torrentclick://add?name=CustomName.torrent&data=" + b64Std
	m1, err := Parse(u1)
	if err != nil {
		t.Fatalf("unexpected error parsing torrentclick query: %v", err)
	}
	if m1.DisplayName != "testfile.iso" { // info dict name takes precedence or CustomName if empty
		t.Errorf("expected display name testfile.iso, got %s", m1.DisplayName)
	}
	if m1.TorrentPath != "CustomName.torrent" {
		t.Errorf("expected TorrentPath CustomName.torrent, got %s", m1.TorrentPath)
	}
	if m1.CustomParams["torrent_file_base64"] == "" {
		t.Errorf("expected torrent_file_base64 in CustomParams")
	}

	// 2. Test torrent://add?data=... with source URL
	u2 := "torrent://add?data=" + b64Std + "&url=https%3A%2F%2Ftracker.org%2Fdl.php%3Fid%3D123"
	m2, err := Parse(u2)
	if err != nil {
		t.Fatalf("unexpected error parsing torrent query: %v", err)
	}
	if m2.CustomParams["source_url"] != "https://tracker.org/dl.php?id=123" {
		t.Errorf("expected source_url to be stored in CustomParams, got %s", m2.CustomParams["source_url"])
	}

	// 3. Test torrentclick://base64,... data URI format
	u3 := "torrentclick://base64," + b64Std
	m3, err := Parse(u3)
	if err != nil {
		t.Fatalf("unexpected error parsing data URI format: %v", err)
	}
	if m3.InfoHash != m1.InfoHash {
		t.Errorf("expected matching info hashes")
	}

	// 4. Test torrentclick:// with embedded magnet link in query
	u4 := "torrentclick://add?magnet=magnet%3A%3Fxt%3Durn%3Abtih%3Ac12fe1c06bba254a9dc9f519b335380dc1f74c26%26dn%3DTestEmbedded"
	m4, err := Parse(u4)
	if err != nil {
		t.Fatalf("unexpected error parsing embedded magnet in query: %v", err)
	}
	if m4.DisplayName != "TestEmbedded" {
		t.Errorf("expected DisplayName TestEmbedded, got %s", m4.DisplayName)
	}

	// 5. Test invalid base64 payload
	_, err = Parse("torrentclick://add?data=not-valid-base64!@#$")
	if err == nil {
		t.Errorf("expected error for invalid base64")
	}

	// 6. Test missing data payload
	_, err = Parse("torrentclick://add?foo=bar")
	if err == nil {
		t.Errorf("expected error for missing data payload")
	}
}

func TestToTemplateContext(t *testing.T) {
	raw := "magnet:?xt=urn:btih:1234567890abcdef&dn=Test+Movie"
	m, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := map[string]string{
		"DownloadDir": "/volume1/downloads",
		"Category":    "movies",
	}
	ctx := m.ToTemplateContext(cfg, "session-xyz")

	if ctx["DisplayName"] != "Test Movie" {
		t.Errorf("context display name mismatch")
	}
	if ctx["SessionID"] != "session-xyz" {
		t.Errorf("context session mismatch")
	}
	cfgMap := ctx["Config"].(map[string]string)
	if cfgMap["Category"] != "movies" {
		t.Errorf("expected category 'movies', got %s", cfgMap["Category"])
	}
}
