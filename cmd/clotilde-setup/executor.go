package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CommandSpec is a single external command invocation.
type CommandSpec struct {
	Stage          string
	Name           string
	Args           []string
	Stdin          string
	SensitiveStdin bool
	AllowFailure   bool
}

// CommandRunner runs external commands. It is intentionally small so tests can
// assert the deployment plan without touching Docker or Google Cloud.
type CommandRunner interface {
	LookPath(name string) error
	Run(ctx context.Context, spec CommandSpec) (string, error)
	Commands() []LoggedCommand
}

// ShellRunner executes real commands.
type ShellRunner struct {
	commands []LoggedCommand
}

func (r *ShellRunner) LookPath(name string) error {
	_, err := exec.LookPath(name)
	return err
}

func (r *ShellRunner) Run(ctx context.Context, spec CommandSpec) (string, error) {
	r.commands = append(r.commands, LoggedCommand{
		Stage: spec.Stage,
		Name:  spec.Name,
		Args:  append([]string(nil), spec.Args...),
	})

	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	if spec.Stdin != "" {
		cmd.Stdin = strings.NewReader(spec.Stdin)
	}

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if err != nil && !spec.AllowFailure {
		return output.String(), fmt.Errorf("%s %s failed: %w\n%s", spec.Name, strings.Join(spec.Args, " "), err, output.String())
	}
	return output.String(), err
}

func (r *ShellRunner) Commands() []LoggedCommand {
	return append([]LoggedCommand(nil), r.commands...)
}

// DryRunRunner records commands and returns deterministic outputs.
type DryRunRunner struct {
	ProjectID   string
	Region      string
	ServiceName string
	commands    []LoggedCommand
}

func (r *DryRunRunner) LookPath(name string) error {
	return nil
}

func (r *DryRunRunner) Run(_ context.Context, spec CommandSpec) (string, error) {
	r.commands = append(r.commands, LoggedCommand{
		Stage: spec.Stage,
		Name:  spec.Name,
		Args:  append([]string(nil), spec.Args...),
	})

	if spec.Name == "gcloud" && len(spec.Args) >= 3 {
		joined := strings.Join(spec.Args, " ")
		switch {
		case strings.HasPrefix(joined, "config get-value project"):
			return r.ProjectID + "\n", nil
		case strings.HasPrefix(joined, "projects describe"):
			return "123456789\n", nil
		case strings.HasPrefix(joined, "run services describe"):
			return fmt.Sprintf("https://%s-dry-run.example.run.app\n", r.ServiceName), nil
		case strings.HasPrefix(joined, "artifacts repositories describe"):
			return "", fmt.Errorf("dry-run: repository would be created")
		case strings.HasPrefix(joined, "secrets describe"):
			return "", fmt.Errorf("dry-run: secret would be created")
		}
	}

	return "dry-run\n", nil
}

func (r *DryRunRunner) Commands() []LoggedCommand {
	return append([]LoggedCommand(nil), r.commands...)
}
