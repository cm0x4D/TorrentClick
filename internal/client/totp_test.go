package client

import (
	"testing"
	"time"
)

func TestGenerateTOTP(t *testing.T) {
	// Standard test secret "12345678901234567890" in base32: GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	// T0 = 59s -> step 1 -> code 287082
	testTime := time.Unix(59, 0)
	code, err := GenerateTOTP(secret, testTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if code != "287082" {
		t.Errorf("expected 287082, got %s", code)
	}
}
