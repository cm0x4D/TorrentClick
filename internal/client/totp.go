package client

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// GenerateTOTP computes a 6-digit TOTP code using RFC 6238 from a base32 secret.
func GenerateTOTP(secret string, t time.Time) (string, error) {
	cleanSecret := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	// Add padding if missing
	if m := len(cleanSecret) % 8; m != 0 {
		cleanSecret += strings.Repeat("=", 8-m)
	}

	key, err := base32.StdEncoding.DecodeString(cleanSecret)
	if err != nil {
		return "", fmt.Errorf("invalid base32 TOTP secret: %w", err)
	}

	interval := uint64(t.Unix() / 30)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], interval)

	h := hmac.New(sha1.New, key)
	h.Write(buf[:])
	hash := h.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	binaryCode := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff
	code := binaryCode % 1000000

	return fmt.Sprintf("%06d", code), nil
}
