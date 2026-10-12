package magnet

import (
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

// Parse extracts structured information from either a magnet: URI or a .torrent file path.
func Parse(rawInput string) (*Magnet, error) {
	trimmed := strings.TrimSpace(rawInput)
	if trimmed == "" {
		return nil, fmt.Errorf("empty magnet or torrent input")
	}

	// 1. Magnet link scheme
	if strings.HasPrefix(strings.ToLower(trimmed), "magnet:") {
		return parseMagnetURI(trimmed)
	}

	// 2. File path or file:// URL (.torrent)
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
