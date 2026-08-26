package agent

import (
	"strings"
	"testing"

	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/msauth"
)

func TestSignInPromptIsReadableFromAcrossTheRoom(t *testing.T) {
	var out strings.Builder
	writeSignInPrompt(&out, &msauth.DeviceCode{
		UserCode:        "F8B2X9AQ",
		VerificationURI: "https://microsoft.com/link",
		ExpiresIn:       900,
	})

	rendered := out.String()
	for _, want := range []string{"F8B2X9AQ", "https://microsoft.com/link", "expires in 15m"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("prompt is missing %q:\n%s", want, rendered)
		}
	}

	var width int
	for _, line := range strings.Split(strings.TrimSpace(rendered), "\n") {
		if width == 0 {
			width = len(line)
		}
		if len(line) != width {
			t.Fatalf("box is ragged, line %q is %d not %d", line, len(line), width)
		}
	}
}

func TestCenterPads(t *testing.T) {
	if got := center("ab", 6); got != "  ab  " {
		t.Errorf("expected padding on both sides, got %q", got)
	}
	if got := center("toolongforthis", 4); got != "toolongforthis" {
		t.Errorf("oversized text should pass through, got %q", got)
	}
}
