package chat

import (
	"strings"
	"testing"
)

func TestSplitMessageShort(t *testing.T) {
	parts := splitMessage("hello", 4096)
	if len(parts) != 1 || parts[0] != "hello" {
		t.Errorf("expected single part %q, got %v", "hello", parts)
	}
}

func TestSplitMessageExactBoundary(t *testing.T) {
	msg := strings.Repeat("a", 4096)
	parts := splitMessage(msg, 4096)
	if len(parts) != 1 || parts[0] != msg {
		t.Errorf("expected single part of len 4096, got %d parts", len(parts))
	}
}

func TestSplitMessageNewlineSplit(t *testing.T) {
	// Build a message that exceeds maxLen, with a newline near the boundary.
	line1 := strings.Repeat("a", 3000) + "\n"
	line2 := strings.Repeat("b", 2000)
	msg := line1 + line2

	parts := splitMessage(msg, 4096)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if parts[0] != line1 {
		t.Errorf("part[0] len = %d, want %d (should split at newline)", len(parts[0]), len(line1))
	}
	if parts[1] != line2 {
		t.Errorf("part[1] = %q, want line2", parts[1])
	}
}

func TestSplitMessageNoNewlineFallback(t *testing.T) {
	// No newlines at all — must hard-split at maxLen.
	msg := strings.Repeat("x", 5000)
	parts := splitMessage(msg, 4096)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if len(parts[0]) != 4096 {
		t.Errorf("part[0] len = %d, want 4096", len(parts[0]))
	}
	if len(parts[1]) != 904 {
		t.Errorf("part[1] len = %d, want 904", len(parts[1]))
	}
}

func TestSplitMessageEmpty(t *testing.T) {
	parts := splitMessage("", 4096)
	if parts != nil {
		t.Errorf("expected nil for empty string, got %v", parts)
	}
}

func TestIsAllowedUserPasses(t *testing.T) {
	if !isAllowed(12345, []int64{12345, 67890}) {
		t.Error("expected user 12345 to be allowed")
	}
}

func TestIsAllowedUserRejected(t *testing.T) {
	if isAllowed(99999, []int64{12345, 67890}) {
		t.Error("expected user 99999 to be rejected")
	}
}

func TestIsAllowedEmptyListRejectsAll(t *testing.T) {
	if isAllowed(12345, nil) {
		t.Error("expected empty allowlist to reject all users")
	}
	if isAllowed(12345, []int64{}) {
		t.Error("expected empty allowlist to reject all users")
	}
}
