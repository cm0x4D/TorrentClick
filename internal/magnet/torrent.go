package magnet

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type bencodeParser struct {
	data      []byte
	pos       int
	infoStart int
	infoEnd   int
}

func (p *bencodeParser) parse() (interface{}, error) {
	if p.pos >= len(p.data) {
		return nil, errors.New("unexpected end of bencode data")
	}

	b := p.data[p.pos]
	switch {
	case b >= '0' && b <= '9':
		return p.parseString()
	case b == 'i':
		return p.parseInteger()
	case b == 'l':
		return p.parseList()
	case b == 'd':
		return p.parseDictionary()
	default:
		return nil, fmt.Errorf("invalid bencode byte '%c' at offset %d", b, p.pos)
	}
}

func (p *bencodeParser) parseString() ([]byte, error) {
	colonIdx := -1
	for i := p.pos; i < len(p.data); i++ {
		if p.data[i] == ':' {
			colonIdx = i
			break
		}
		if p.data[i] < '0' || p.data[i] > '9' {
			return nil, fmt.Errorf("invalid character in string length '%c' at offset %d", p.data[i], i)
		}
	}
	if colonIdx == -1 {
		return nil, errors.New("missing colon in string length")
	}

	lenStr := string(p.data[p.pos:colonIdx])
	length, err := strconv.Atoi(lenStr)
	if err != nil || length < 0 {
		return nil, fmt.Errorf("invalid string length '%s'", lenStr)
	}

	start := colonIdx + 1
	end := start + length
	if end > len(p.data) {
		return nil, fmt.Errorf("string length %d exceeds remaining buffer size", length)
	}

	p.pos = end
	return p.data[start:end], nil
}

func (p *bencodeParser) parseInteger() (int64, error) {
	p.pos++ // skip 'i'
	eIdx := -1
	for i := p.pos; i < len(p.data); i++ {
		if p.data[i] == 'e' {
			eIdx = i
			break
		}
	}
	if eIdx == -1 {
		return 0, errors.New("missing terminating 'e' in integer")
	}

	intStr := string(p.data[p.pos:eIdx])
	val, err := strconv.ParseInt(intStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer '%s'", intStr)
	}

	p.pos = eIdx + 1
	return val, nil
}

func (p *bencodeParser) parseList() ([]interface{}, error) {
	p.pos++ // skip 'l'
	list := make([]interface{}, 0)
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		elem, err := p.parse()
		if err != nil {
			return nil, err
		}
		list = append(list, elem)
	}
	if p.pos >= len(p.data) {
		return nil, errors.New("unterminated list (missing 'e')")
	}
	p.pos++ // skip 'e'
	return list, nil
}

func (p *bencodeParser) parseDictionary() (map[string]interface{}, error) {
	p.pos++ // skip 'd'
	dict := make(map[string]interface{})
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		keyBytes, err := p.parseString()
		if err != nil {
			return nil, fmt.Errorf("failed to parse dictionary key: %w", err)
		}
		key := string(keyBytes)

		valStart := p.pos
		val, err := p.parse()
		if err != nil {
			return nil, fmt.Errorf("failed to parse value for key '%s': %w", key, err)
		}
		valEnd := p.pos

		if key == "info" {
			p.infoStart = valStart
			p.infoEnd = valEnd
		}

		dict[key] = val
	}
	if p.pos >= len(p.data) {
		return nil, errors.New("unterminated dictionary (missing 'e')")
	}
	p.pos++ // skip 'e'
	return dict, nil
}

// ParseTorrentBytes parses raw .torrent file bytes and extracts metadata into a Magnet representation.
func ParseTorrentBytes(data []byte, sourcePath string) (*Magnet, error) {
	if len(data) == 0 {
		return nil, errors.New("empty torrent data")
	}

	parser := &bencodeParser{data: data}
	rootVal, err := parser.parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse bencode: %w", err)
	}

	rootDict, ok := rootVal.(map[string]interface{})
	if !ok {
		return nil, errors.New("torrent root is not a dictionary")
	}

	if parser.infoStart == 0 && parser.infoEnd == 0 {
		return nil, errors.New("torrent is missing 'info' dictionary")
	}

	infoBytes := data[parser.infoStart:parser.infoEnd]
	sum := sha1.Sum(infoBytes)
	infoHash := hex.EncodeToString(sum[:])

	infoDict, ok := rootDict["info"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid 'info' structure in torrent")
	}

	var displayName string
	if nameBytes, ok := infoDict["name"].([]byte); ok {
		displayName = string(nameBytes)
	}

	var totalSize int64
	if lengthVal, ok := infoDict["length"].(int64); ok {
		totalSize = lengthVal
	} else if filesVal, ok := infoDict["files"].([]interface{}); ok {
		for _, f := range filesVal {
			if fDict, ok := f.(map[string]interface{}); ok {
				if fLen, ok := fDict["length"].(int64); ok {
					totalSize += fLen
				}
			}
		}
	}

	trackers := make([]string, 0)
	trackerSeen := make(map[string]bool)

	if annBytes, ok := rootDict["announce"].([]byte); ok && len(annBytes) > 0 {
		ann := string(annBytes)
		trackers = append(trackers, ann)
		trackerSeen[ann] = true
	}

	if annList, ok := rootDict["announce-list"].([]interface{}); ok {
		for _, tier := range annList {
			if tierList, ok := tier.([]interface{}); ok {
				for _, trElem := range tierList {
					if trBytes, ok := trElem.([]byte); ok && len(trBytes) > 0 {
						tr := string(trBytes)
						if !trackerSeen[tr] {
							trackers = append(trackers, tr)
							trackerSeen[tr] = true
						}
					}
				}
			}
		}
	}

	if displayName == "" {
		displayName = "Torrent-" + infoHash[:min(8, len(infoHash))]
	}

	// Synthesize standard magnet URI
	magnetURI := "magnet:?xt=urn:btih:" + infoHash
	if displayName != "" {
		magnetURI += "&dn=" + url.QueryEscape(displayName)
	}
	for _, tr := range trackers {
		magnetURI += "&tr=" + url.QueryEscape(tr)
	}

	m := &Magnet{
		Raw:          magnetURI,
		RawEscaped:   url.QueryEscape(magnetURI),
		DisplayName:  displayName,
		InfoHash:     infoHash,
		InfoHashType: "btih",
		Trackers:     trackers,
		ExactSources: make([]string, 0),
		Keywords:     make([]string, 0),
		Size:         fmt.Sprintf("%d", totalSize),
		TorrentPath:  sourcePath,
		CustomParams: map[string]string{
			"torrent_file_path":   sourcePath,
			"torrent_file_base64": base64.StdEncoding.EncodeToString(data),
		},
	}

	return m, nil
}

// ParseTorrentFile reads and parses a .torrent file from disk.
func ParseTorrentFile(filePath string) (*Magnet, error) {
	cleanPath := strings.TrimSpace(filePath)
	if strings.HasPrefix(cleanPath, "file://") {
		u, err := url.Parse(cleanPath)
		if err == nil {
			cleanPath = u.Path
		} else {
			cleanPath = strings.TrimPrefix(cleanPath, "file://")
		}
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read torrent file: %w", err)
	}

	return ParseTorrentBytes(data, cleanPath)
}
