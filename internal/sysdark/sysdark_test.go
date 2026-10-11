package sysdark

import (
	"os/exec"
	"strings"
	"testing"
)

func TestIsDarkMode(t *testing.T) {
	out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
	expectedDark := err == nil && strings.TrimSpace(string(out)) == "Dark"

	actual := IsDarkMode()
	if actual != expectedDark {
		t.Errorf("IsDarkMode() = %v, expected from defaults %v", actual, expectedDark)
	}
}
