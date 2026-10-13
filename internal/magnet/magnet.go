package magnet

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Magnet represents parsed information from a magnet URI or .torrent file.
type Magnet struct {
	Raw           string            `json:"raw"`
	RawEscaped    string            `json:"raw_escaped"`
	DisplayName   string            `json:"display_name"`
	InfoHash      string            `json:"info_hash"`
	InfoHashType  string            `json:"info_hash_type"`
	Trackers      []string          `json:"trackers"`
	ExactSources  []string          `json:"exact_sources"`
	Keywords      []string          `json:"keywords"`
	Size          string            `json:"size"`
	TorrentPath   string            `json:"torrent_path,omitempty"`
	CustomParams  map[string]string `json:"custom_params"`
}

// Parse extracts structured information from either a magnet: URI, a torrentclick:/torrent: scheme, or a .torrent file path.
func Parse(rawInput string) (*Magnet, error) {
	trimmed := strings.TrimSpace(rawInput)
	if trimmed == "" {
		return nil, fmt.Errorf("empty magnet or torrent input")
	}

	// 1. Magnet link scheme
	if strings.HasPrefix(strings.ToLower(trimmed), "magnet:") {
		return parseMagnetURI(trimmed)
	}

	// 2. Custom torrentclick: or torrent: URL scheme with base64 payload or query params
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "torrentclick:") || strings.HasPrefix(lower, "torrent:") {
		return parseTorrentScheme(trimmed)
	}

	// 3. File path or file:// URL (.torrent)
	cleanPath := trimmed
	if strings.HasPrefix(cleanPath, "file://") {
		u, err := url.Parse(cleanPath)
		if err == nil {
			cleanPath = u.Path
		} else {
			cleanPath = strings.TrimPrefix(cleanPath, "file://")
		}
	}

	if strings.HasSuffix(strings.ToLower(cleanPath), ".torrent") {
		return ParseTorrentFile(cleanPath)
	}

	// If file exists on filesystem, attempt parsing as torrent
	if fi, err := os.Stat(cleanPath); err == nil && !fi.IsDir() {
		m, err := ParseTorrentFile(cleanPath)
		if err == nil {
			return m, nil
		}
	}

	return nil, fmt.Errorf("invalid input: expected magnet: URI or .torrent file path")
}

func parseMagnetURI(trimmed string) (*Magnet, error) {

	u, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse magnet URI: %w", err)
	}

	queryParams, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parse magnet query string: %w", err)
	}

	m := &Magnet{
		Raw:          trimmed,
		RawEscaped:   url.QueryEscape(trimmed),
		Trackers:     make([]string, 0),
		ExactSources: make([]string, 0),
		Keywords:     make([]string, 0),
		CustomParams: make(map[string]string),
	}

	for key, values := range queryParams {
		switch strings.ToLower(key) {
		case "xt":
			for _, val := range values {
				valLower := strings.ToLower(val)
				if strings.HasPrefix(valLower, "urn:btih:") {
					m.InfoHash = val[len("urn:btih:"):]
					m.InfoHashType = "btih"
				} else if strings.HasPrefix(valLower, "urn:btmh:") {
					m.InfoHash = val[len("urn:btmh:"):]
					m.InfoHashType = "btmh"
				} else if m.InfoHash == "" {
					m.InfoHash = val
				}
			}
		case "dn":
			if len(values) > 0 {
				m.DisplayName = values[0]
			}
		case "tr":
			m.Trackers = append(m.Trackers, values...)
		case "xs":
			m.ExactSources = append(m.ExactSources, values...)
		case "kt":
			m.Keywords = append(m.Keywords, values...)
		case "xl":
			if len(values) > 0 {
				m.Size = values[0]
			}
		default:
			if len(values) > 0 {
				m.CustomParams[key] = values[0]
			}
		}
	}

	if m.DisplayName == "" {
		if m.InfoHash != "" {
			m.DisplayName = "Torrent-" + m.InfoHash[:min(8, len(m.InfoHash))]
		} else {
			m.DisplayName = "Unknown Torrent"
		}
	}

	return m, nil
}

// ToTemplateContext prepares a map suitable for Go text/template execution.
func (m *Magnet) ToTemplateContext(cfgParams map[string]string, sessionID string) map[string]interface{} {
	ctx := map[string]interface{}{
		"Magnet":        m.Raw,
		"MagnetEscaped": m.RawEscaped,
		"InfoHash":      m.InfoHash,
		"InfoHashType":  m.InfoHashType,
		"DisplayName":   m.DisplayName,
		"Name":          m.DisplayName,
		"Trackers":      m.Trackers,
		"Size":          m.Size,
		"SessionID":     sessionID,
		"Config":        cfgParams,
		"TorrentPath":   m.TorrentPath,
		"TorrentBase64": m.CustomParams["torrent_file_base64"],
	}
	return ctx
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseTorrentScheme parses custom torrentclick: and torrent: URL schemes.
// Supported formats:
// 1. Query parameters: torrentclick://add?data=<base64>&name=<filename>&url=<source_url>
//    or torrent://add?data=<base64>
//    or torrentclick:?data=<base64>
// 2. Data payload: torrentclick://base64,<payload> or torrent://base64,<payload>
// 3. Raw payload: torrentclick://<base64> or torrent://<base64>
func parseTorrentScheme(rawInput string) (*Magnet, error) {
	trimmed := strings.TrimSpace(rawInput)

	var b64Data string
	var name string
	var sourceURL string

	// Check if URL has query parameters
	if qIdx := strings.Index(trimmed, "?"); qIdx != -1 {
		queryString := trimmed[qIdx+1:]
		values, err := url.ParseQuery(queryString)
		if err == nil {
			if v := values.Get("data"); v != "" {
				b64Data = v
			} else if v := values.Get("b64"); v != "" {
				b64Data = v
			} else if v := values.Get("payload"); v != "" {
				b64Data = v
			}

			name = values.Get("name")
			if name == "" {
				name = values.Get("dn")
			}
			sourceURL = values.Get("url")

			// Check if a magnet link was passed inside the query
			if b64Data == "" {
				if magParam := values.Get("magnet"); magParam != "" {
					return parseMagnetURI(magParam)
				}
			}
		}
	}

	// If no query parameter data was found, check data payload formats
	if b64Data == "" {
		// Strip scheme prefix (e.g. torrentclick://, torrent://, torrentclick:, torrent:)
		payload := trimmed
		if idx := strings.Index(payload, ":"); idx != -1 {
			payload = payload[idx+1:]
		}
		payload = strings.TrimPrefix(payload, "//")
		if idx := strings.Index(payload, "base64,"); idx != -1 {
			payload = payload[idx+7:]
		}
		// Strip trailing query or fragments if any
		if idx := strings.IndexAny(payload, "?#"); idx != -1 {
			payload = payload[:idx]
		}
		b64Data = payload
	}

	if b64Data == "" {
		return nil, errors.New("torrent scheme missing data payload")
	}

	rawBytes, err := decodeBase64(b64Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 torrent data: %w", err)
	}

	sourcePath := name
	if sourcePath == "" {
		sourcePath = "remote.torrent"
	}

	mag, err := ParseTorrentBytes(rawBytes, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decoded torrent bytes: %w", err)
	}

	if name != "" {
		cleanName := strings.TrimSuffix(name, ".torrent")
		if mag.DisplayName == "" || strings.HasPrefix(mag.DisplayName, "Torrent-") {
			mag.DisplayName = cleanName
		}
	}
	if sourceURL != "" {
		mag.CustomParams["source_url"] = sourceURL
	}

	return mag, nil
}

func decodeBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "base64,"); idx != -1 {
		s = s[idx+7:]
	}
	// Query decoding may turn '+' into spaces if not URL-encoded
	s = strings.ReplaceAll(s, " ", "+")

	// 1. Try standard base64 with padding
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) > 0 {
		return b, nil
	}
	// 2. Try raw standard base64 (no padding)
	if b, err := base64.RawStdEncoding.DecodeString(s); err == nil && len(b) > 0 {
		return b, nil
	}
	// 3. Try URL-safe base64
	if b, err := base64.URLEncoding.DecodeString(s); err == nil && len(b) > 0 {
		return b, nil
	}
	// 4. Try raw URL-safe base64 (no padding)
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil && len(b) > 0 {
		return b, nil
	}

	return nil, errors.New("invalid base64 encoding")
}
