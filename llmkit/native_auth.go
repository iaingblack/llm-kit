package llmkit

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"time"
)

const defaultNativeAuthCheckTimeout = 5 * time.Second

type NativeAuthStatus struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Command       string `json:"command"`
	Installed     bool   `json:"installed"`
	Authenticated bool   `json:"authenticated"`
	Status        string `json:"status"`
}

type NativeAuthCommand struct {
	ID      string
	Label   string
	Binary  string
	Args    []string
	Command string
}

type NativeAuthChecker struct {
	Commands []NativeAuthCommand
	LookPath func(string) (string, error)
	Run      func(context.Context, string, ...string) error
	Timeout  time.Duration
}

var DefaultNativeAuthCommands = []NativeAuthCommand{
	{
		ID:      "codex",
		Label:   "ChatGPT (Codex CLI)",
		Binary:  "codex",
		Args:    []string{"login", "status"},
		Command: "codex login status",
	},
	{
		ID:      "claude",
		Label:   "Claude Code",
		Binary:  "claude",
		Args:    []string{"auth", "status"},
		Command: "claude auth status",
	},
}

func (c NativeAuthChecker) CheckAll(ctx context.Context) []NativeAuthStatus {
	commands := c.commands()
	statuses := make([]NativeAuthStatus, 0, len(commands))
	for _, command := range commands {
		statuses = append(statuses, c.Check(ctx, command))
	}
	return statuses
}

func (c NativeAuthChecker) Check(ctx context.Context, command NativeAuthCommand) NativeAuthStatus {
	status := NativeAuthStatus{
		ID:      command.ID,
		Label:   command.Label,
		Command: command.Command,
	}

	path, err := c.lookPath()(command.Binary)
	if err != nil {
		status.Status = "not_installed"
		return status
	}
	status.Installed = true

	checkCtx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	if err := c.run()(checkCtx, path, command.Args...); err != nil {
		if errors.Is(checkCtx.Err(), context.DeadlineExceeded) {
			status.Status = "timeout"
			return status
		}
		status.Status = "not_authenticated"
		return status
	}

	status.Authenticated = true
	status.Status = "authenticated"
	return status
}

func (c NativeAuthChecker) commands() []NativeAuthCommand {
	if len(c.Commands) > 0 {
		return c.Commands
	}
	return DefaultNativeAuthCommands
}

func (c NativeAuthChecker) lookPath() func(string) (string, error) {
	if c.LookPath != nil {
		return c.LookPath
	}
	return exec.LookPath
}

func (c NativeAuthChecker) run() func(context.Context, string, ...string) error {
	if c.Run != nil {
		return c.Run
	}
	return runNativeAuthCommand
}

func (c NativeAuthChecker) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return defaultNativeAuthCheckTimeout
}

func runNativeAuthCommand(ctx context.Context, path string, args ...string) error {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}
