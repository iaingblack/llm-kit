package llmkit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const defaultNativeLLMTimeout = 5 * time.Minute

type NativeCompletionRequest struct {
	Provider string
	Model    string
	Messages any
	Tools    any
	Header   string
}

type NativeCompletionResponse struct {
	Content   string           `json:"content"`
	ToolCalls []NativeToolCall `json:"tool_calls"`
}

type NativeToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type NativeCLI struct {
	LookPath   func(string) (string, error)
	RunCommand func(context.Context, string, []string, string) (string, string, error)
	Timeout    time.Duration
	WorkDir    string
}

func (c NativeCLI) Complete(ctx context.Context, req NativeCompletionRequest) (*NativeCompletionResponse, error) {
	prompt, err := BuildNativePrompt(req)
	if err != nil {
		return nil, err
	}

	var raw string
	switch req.Provider {
	case ProviderCodexCLI:
		raw, err = c.runCodex(ctx, req.Model, prompt)
	case ProviderClaudeCodeCLI:
		raw, err = c.runClaude(ctx, req.Model, prompt)
	default:
		err = fmt.Errorf("unsupported native provider: %s", req.Provider)
	}
	if err != nil {
		return nil, err
	}
	return ParseNativeResponse(raw)
}

func BuildNativePrompt(req NativeCompletionRequest) (string, error) {
	toolData, err := json.MarshalIndent(req.Tools, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal native tools: %w", err)
	}
	messageData, err := json.MarshalIndent(req.Messages, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal native messages: %w", err)
	}

	header := strings.TrimSpace(req.Header)
	if header == "" {
		header = "You are the model backend for an application chat assistant."
	}

	return fmt.Sprintf(`%s

Return ONLY a JSON object matching this schema:
{
  "content": "User-visible assistant text. Include a short explanation before tool calls.",
  "tool_calls": [
    {
      "name": "one of the available tool names",
      "arguments": "{\"name\":\"resource-name\"}"
    }
  ]
}

Use tool_calls only when the user explicitly asks the application to perform an action or fetch information not already in the current state.
If no tool is needed, return "tool_calls": [].
Do not call shell commands, edit files, or use native CLI tools. The host application will execute only the JSON tool calls returned here.
Do not invent tools. The arguments field MUST be a JSON-encoded object string matching the selected tool's parameters.
For an empty-argument tool, use "arguments": "{}".

Available tools:
%s

Conversation messages:
%s
`, header, string(toolData), string(messageData)), nil
}

func (c NativeCLI) runCodex(ctx context.Context, model string, prompt string) (string, error) {
	path, err := c.lookPath()("codex")
	if err != nil {
		return "", fmt.Errorf("Codex CLI is not installed or not on PATH")
	}

	schemaPath, cleanupSchema, err := writeNativeSchemaTempFile()
	if err != nil {
		return "", err
	}
	defer cleanupSchema()

	outputFile, err := os.CreateTemp("", "llm-kit-codex-output-*.txt")
	if err != nil {
		return "", fmt.Errorf("create Codex output file: %w", err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputPath)

	args := []string{
		"exec",
		"--skip-git-repo-check",
		"--ephemeral",
		"--cd", c.workDir(),
		"--sandbox", "read-only",
		"--output-schema", schemaPath,
		"--output-last-message", outputPath,
		"--color", "never",
	}
	if model = strings.TrimSpace(model); model != "" && model != "codex-default" {
		args = append(args, "--model", model)
	}
	args = append(args, "-")

	runCtx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	stdout, stderr, err := c.runCommand()(runCtx, path, args, prompt)
	if err != nil {
		return "", nativeCommandError("Codex CLI", runCtx, stdout, stderr, err)
	}

	if data, readErr := os.ReadFile(outputPath); readErr == nil && len(bytes.TrimSpace(data)) > 0 {
		return string(data), nil
	}
	return stdout, nil
}

func (c NativeCLI) runClaude(ctx context.Context, model string, prompt string) (string, error) {
	path, err := c.lookPath()("claude")
	if err != nil {
		return "", fmt.Errorf("Claude Code CLI is not installed or not on PATH")
	}

	args := []string{
		"--print",
		"--output-format", "json",
		"--input-format", "text",
		"--tools", "",
		"--permission-mode", "dontAsk",
		"--no-session-persistence",
		"--json-schema", NativeSchemaJSON(),
	}
	if model = strings.TrimSpace(model); model != "" && model != "claude-default" {
		args = append(args, "--model", model)
	}

	runCtx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	stdout, stderr, err := c.runCommand()(runCtx, path, args, prompt)
	if err != nil {
		return "", nativeCommandError("Claude Code CLI", runCtx, stdout, stderr, err)
	}
	return stdout, nil
}

func ParseNativeResponse(raw string) (*NativeCompletionResponse, error) {
	payload := strings.TrimSpace(raw)
	if payload == "" {
		return nil, fmt.Errorf("native CLI returned an empty response")
	}

	if resp, ok := parseNativeObject(payload); ok {
		return resp, nil
	}

	var wrapper map[string]any
	if err := json.Unmarshal([]byte(payload), &wrapper); err == nil {
		for _, key := range []string{"result", "content", "message"} {
			if nested, ok := wrapper[key].(string); ok {
				if resp, nestedOK := parseNativeObject(strings.TrimSpace(nested)); nestedOK {
					return resp, nil
				}
			}
		}
	}

	start := strings.Index(payload, "{")
	end := strings.LastIndex(payload, "}")
	if start >= 0 && end > start {
		if resp, ok := parseNativeObject(payload[start : end+1]); ok {
			return resp, nil
		}
	}

	return nil, fmt.Errorf("native CLI returned invalid structured response")
}

func NativeToolArgumentsJSON(raw json.RawMessage) (string, error) {
	args := bytes.TrimSpace(raw)
	if len(args) == 0 || bytes.Equal(args, []byte("null")) {
		return "{}", nil
	}
	if len(args) > 0 && args[0] == '"' {
		var encoded string
		if err := json.Unmarshal(args, &encoded); err != nil {
			return "", fmt.Errorf("parse native tool arguments string: %w", err)
		}
		args = []byte(strings.TrimSpace(encoded))
	}
	if len(args) == 0 {
		return "{}", nil
	}
	if !json.Valid(args) {
		return "", fmt.Errorf("native tool arguments are not valid JSON")
	}
	var obj map[string]any
	if err := json.Unmarshal(args, &obj); err != nil {
		return "", fmt.Errorf("native tool arguments must be a JSON object: %w", err)
	}
	return string(args), nil
}

func NativeSchemaJSON() string {
	return `{
  "type": "object",
  "additionalProperties": false,
  "required": ["content", "tool_calls"],
  "properties": {
    "content": {
      "type": "string"
    },
    "tool_calls": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["name", "arguments"],
        "properties": {
          "name": {
            "type": "string"
          },
          "arguments": {
            "type": "string",
            "description": "JSON-encoded object string for the tool arguments, such as \"{}\" or \"{\\\"name\\\":\\\"vm1\\\"}\""
          }
        }
      }
    }
  }
}`
}

func parseNativeObject(payload string) (*NativeCompletionResponse, bool) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &fields); err != nil {
		return nil, false
	}
	if _, ok := fields["content"]; !ok {
		return nil, false
	}
	if _, ok := fields["tool_calls"]; !ok {
		return nil, false
	}

	var resp NativeCompletionResponse
	if err := json.Unmarshal([]byte(payload), &resp); err != nil {
		return nil, false
	}
	if resp.ToolCalls == nil {
		resp.ToolCalls = []NativeToolCall{}
	}
	return &resp, true
}

func writeNativeSchemaTempFile() (string, func(), error) {
	f, err := os.CreateTemp("", "llm-kit-native-schema-*.json")
	if err != nil {
		return "", nil, fmt.Errorf("create native schema file: %w", err)
	}
	path := f.Name()
	cleanup := func() { _ = os.Remove(path) }
	if _, err := io.WriteString(f, NativeSchemaJSON()); err != nil {
		f.Close()
		cleanup()
		return "", nil, fmt.Errorf("write native schema file: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close native schema file: %w", err)
	}
	return path, cleanup, nil
}

func nativeCommandError(label string, ctx context.Context, stdout, stderr string, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%s timed out", label)
	}
	detail := strings.TrimSpace(stderr)
	if detail == "" {
		detail = strings.TrimSpace(stdout)
	}
	if detail == "" {
		detail = err.Error()
	}
	return fmt.Errorf("%s failed: %s", label, truncate(detail, 500))
}

func (c NativeCLI) lookPath() func(string) (string, error) {
	if c.LookPath != nil {
		return c.LookPath
	}
	return exec.LookPath
}

func (c NativeCLI) runCommand() func(context.Context, string, []string, string) (string, string, error) {
	if c.RunCommand != nil {
		return c.RunCommand
	}
	return runNativeCommand
}

func (c NativeCLI) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return defaultNativeLLMTimeout
}

func (c NativeCLI) workDir() string {
	if strings.TrimSpace(c.WorkDir) != "" {
		return filepath.Clean(c.WorkDir)
	}
	return filepath.Clean(os.TempDir())
}

func runNativeCommand(ctx context.Context, path string, args []string, stdin string) (string, string, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = filepath.Clean(os.TempDir())
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
