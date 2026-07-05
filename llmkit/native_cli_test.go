package llmkit

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestNativeCLIParsesCodexToolCalls(t *testing.T) {
	cli := NativeCLI{
		LookPath: func(binary string) (string, error) {
			if binary != "codex" {
				t.Fatalf("binary = %q, want codex", binary)
			}
			return "/usr/local/bin/codex", nil
		},
		RunCommand: func(ctx context.Context, path string, args []string, stdin string) (string, string, error) {
			if !strings.Contains(stdin, "Available tools") {
				t.Fatal("prompt did not include tool definitions")
			}
			writeCodexOutputFile(t, args, `{"content":"Listing.","tool_calls":[{"name":"list_items","arguments":"{}"}]}`)
			return "", "", nil
		},
	}

	resp, err := cli.Complete(context.Background(), NativeCompletionRequest{
		Provider: ProviderCodexCLI,
		Messages: []map[string]string{{"role": "user", "content": "list"}},
		Tools:    []map[string]string{{"name": "list_items"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "Listing." {
		t.Fatalf("Content = %q", resp.Content)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "list_items" {
		t.Fatalf("ToolCalls = %+v", resp.ToolCalls)
	}
}

func TestParseNativeResponseFromClaudeWrapper(t *testing.T) {
	resp, err := ParseNativeResponse(`{"type":"result","result":"{\"content\":\"ok\",\"tool_calls\":[]}"}`)
	if err != nil {
		t.Fatalf("ParseNativeResponse: %v", err)
	}
	if resp.Content != "ok" || len(resp.ToolCalls) != 0 {
		t.Fatalf("response = %+v", resp)
	}
}

func writeCodexOutputFile(t *testing.T, args []string, content string) {
	t.Helper()
	for i, arg := range args {
		if arg == "--output-last-message" && i+1 < len(args) {
			if err := os.WriteFile(args[i+1], []byte(content), 0600); err != nil {
				t.Fatalf("write Codex output: %v", err)
			}
			return
		}
	}
	t.Fatal("Codex args did not include --output-last-message")
}
