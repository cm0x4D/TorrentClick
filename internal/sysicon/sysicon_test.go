package sysicon

import "testing"

func TestNormalize(t *testing.T) {
	if Normalize("both") != ModeBoth {
		t.Errorf("expected ModeBoth for 'both'")
	}
	if Normalize("tray") != ModeTray {
		t.Errorf("expected ModeTray for 'tray'")
	}
	if Normalize("dock") != ModeDock {
		t.Errorf("expected ModeDock for 'dock'")
	}
	if Normalize("") != ModeBoth {
		t.Errorf("expected ModeBoth for empty string")
	}
	if Normalize("invalid") != ModeBoth {
		t.Errorf("expected ModeBoth for invalid string")
	}
}

func TestApplyMode(t *testing.T) {
	// Ensure calls do not panic on any platform
	ApplyMode(ModeBoth)
	ApplyMode(ModeTray)
	ApplyMode(ModeDock)
}
