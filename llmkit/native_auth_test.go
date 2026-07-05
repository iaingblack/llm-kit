package llmkit

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestNativeAuthCheckerMapsInstalledAndAuthenticated(t *testing.T) {
	checker := NativeAuthChecker{
		LookPath: func(binary string) (string, error) {
			switch binary {
			case "codex":
				return "/usr/local/bin/codex", nil
			case "claude":
				return "/usr/local/bin/claude", nil
			default:
				return "", exec.ErrNotFound
			}
		},
		Run: func(ctx context.Context, path string, args ...string) error {
			if path == "/usr/local/bin/codex" {
				return nil
			}
			return errors.New("not logged in")
		},
	}

	statuses := checker.CheckAll(context.Background())
	if len(statuses) != 2 {
		t.Fatalf("len(statuses) = %d, want 2", len(statuses))
	}
	if statuses[0].ID != "codex" || !statuses[0].Installed || !statuses[0].Authenticated || statuses[0].Status != "authenticated" {
		t.Fatalf("codex status = %+v", statuses[0])
	}
	if statuses[1].ID != "claude" || !statuses[1].Installed || statuses[1].Authenticated || statuses[1].Status != "not_authenticated" {
		t.Fatalf("claude status = %+v", statuses[1])
	}
}
