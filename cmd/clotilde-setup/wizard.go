package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	exitOK          = 0
	exitValidation  = 2
	exitProvision   = 3
	exitDeploy      = 4
	exitVerify      = 5
	defaultImageTag = "latest"
)

func runSetup(ctx context.Context, cfg SetupConfig, opts Options, runner CommandRunner, stdin io.Reader, stdout, stderr io.Writer) (SetupResult, int) {
	startedAt := time.Now()
	result := SetupResult{
		OK:           false,
		Secrets:      map[string]string{},
		AdminEnabled: cfg.Admin.Enabled,
		StartedAt:    startedAt,
	}
	progressOut := stdout
	if opts.Output == "json" {
		progressOut = stderr
	}

	applyDefaults(&cfg)
	if opts.ResultPath == "" {
		opts.ResultPath = filepath.Join(".clotilde", "setup-result.json")
	}

	if opts.NonInteractive {
		if cfg.ProjectID == "" {
			project, err := activeProject(ctx, runner)
			if err == nil {
				cfg.ProjectID = strings.TrimSpace(project)
			}
		}
	} else {
		var err error
		cfg, err = collectInteractiveConfig(ctx, cfg, runner, stdin, progressOut)
		if err != nil {
			return failResult(result, err, exitValidation)
		}
	}
	applyDefaults(&cfg)
	result.AdminEnabled = cfg.Admin.Enabled

	if err := validateConfig(cfg); err != nil {
		return failResult(result, err, exitValidation)
	}

	if err := preflight(ctx, cfg, runner, stderr); err != nil {
		return failResult(result, err, exitValidation)
	}

	if !opts.Yes && !opts.NonInteractive {
		if !confirm(stdin, progressOut, "Proceed with provisioning and deployment?", true) {
			return failResult(result, fmt.Errorf("cancelled"), exitValidation)
		}
	}

	if err := provision(ctx, cfg, runner, progressOut); err != nil {
		return failResult(result, err, exitProvision)
	}

	serviceURL, err := deploy(ctx, cfg, opts, runner, progressOut)
	if err != nil {
		return failResult(result, err, exitDeploy)
	}

	if serviceURL == "" {
		serviceURL, _ = describeServiceURL(ctx, cfg, runner)
	}

	if !opts.SkipVerify {
		if err := verify(ctx, serviceURL, cfg, opts, progressOut); err != nil {
			return failResult(result, err, exitVerify)
		}
	}

	result.OK = true
	result.ServiceURL = serviceURL
	result.Secrets = secretInventory(cfg)
	result.NextSteps = nextSteps(serviceURL, cfg)
	result.ResultPath = opts.ResultPath
	result.Commands = runner.Commands()
	result.FinishedAt = time.Now()

	if !opts.DryRun {
		if err := writeResult(opts.ResultPath, result); err != nil {
			return failResult(result, err, exitProvision)
		}
	}

	if opts.Output == "json" {
		printJSON(stdout, result)
	} else {
		printTextResult(stdout, result, opts.DryRun)
	}
	return result, exitOK
}

func failResult(result SetupResult, err error, code int) (SetupResult, int) {
	result.OK = false
	result.Error = err.Error()
	result.FinishedAt = time.Now()
	return result, code
}

func collectInteractiveConfig(ctx context.Context, cfg SetupConfig, runner CommandRunner, stdin io.Reader, stdout io.Writer) (SetupConfig, error) {
	reader := bufio.NewReader(stdin)
	active, _ := activeProject(ctx, runner)
	active = strings.TrimSpace(active)
	if cfg.ProjectID == "" {
		cfg.ProjectID = prompt(reader, stdout, "Google Cloud project ID", active)
	}
	cfg.Region = prompt(reader, stdout, "Region", valueOrDefault(cfg.Region, "us-central1"))
	cfg.ServiceName = prompt(reader, stdout, "Cloud Run service name", valueOrDefault(cfg.ServiceName, "clotilde"))
	cfg.RepoName = prompt(reader, stdout, "Artifact Registry repo name", valueOrDefault(cfg.RepoName, "clotilde-repo"))
	cfg.ImageTag = prompt(reader, stdout, "Image tag", valueOrDefault(cfg.ImageTag, defaultImageTag))
	cfg.LogBufferSize = promptInt(reader, stdout, "Log buffer size", valueOrDefault(strconv.Itoa(cfg.LogBufferSize), "1000"))

	if cfg.OpenAI.SecretName == "" {
		cfg.OpenAI.SecretName = prompt(reader, stdout, "OpenAI secret name", defaultSecretName("clotilde-oai"))
	}
	cfg.OpenAI.Value = promptSecret(stdout, "OpenAI API key")

	if cfg.API.SecretName == "" {
		cfg.API.SecretName = prompt(reader, stdout, "Service API key secret name", defaultSecretName("clotilde-auth"))
	}
	if confirm(reader, stdout, "Generate service API key automatically?", true) {
		cfg.API.Generate = true
	} else {
		cfg.API.Value = promptSecret(stdout, "Service API key")
	}

	cfg.Admin.Enabled = confirm(reader, stdout, "Enable admin dashboard?", true)
	if cfg.Admin.Enabled {
		cfg.Admin.Username = prompt(reader, stdout, "Admin username", valueOrDefault(cfg.Admin.Username, "admin"))
		if cfg.Admin.Password.SecretName == "" {
			cfg.Admin.Password.SecretName = prompt(reader, stdout, "Admin password secret name", defaultSecretName("clotilde-admin"))
		}
		cfg.Admin.Password.Value = promptSecret(stdout, "Admin password")
	}

	cfg.Claude.Enabled = confirm(reader, stdout, "Enable Claude fast responses?", false)
	if cfg.Claude.Enabled {
		cfg.Claude.Secret.SecretName = prompt(reader, stdout, "Claude secret name", defaultSecretName("clotilde-claude"))
		cfg.Claude.Secret.Value = promptSecret(stdout, "Claude API key")
	}

	cfg.Perplexity.Enabled = confirm(reader, stdout, "Enable Perplexity search?", false)
	if cfg.Perplexity.Enabled {
		cfg.Perplexity.Secret.SecretName = prompt(reader, stdout, "Perplexity secret name", defaultSecretName("clotilde-perplexity"))
		cfg.Perplexity.Secret.Value = promptSecret(stdout, "Perplexity API key")
	}

	cfg.ConfigAPI.Enabled = confirm(reader, stdout, "Enable dedicated config API key?", false)
	if cfg.ConfigAPI.Enabled {
		cfg.ConfigAPI.Secret.SecretName = prompt(reader, stdout, "Config API secret name", defaultSecretName("clotilde-config"))
		cfg.ConfigAPI.Secret.Value = promptSecret(stdout, "Config API key")
	}

	return cfg, nil
}

func prompt(reader *bufio.Reader, stdout io.Writer, label, defaultValue string) string {
	if defaultValue != "" {
		fmt.Fprintf(stdout, "%s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(stdout, "%s: ", label)
	}
	value, _ := reader.ReadString('\n')
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue
	}
	return value
}

func promptInt(reader *bufio.Reader, stdout io.Writer, label, defaultValue string) int {
	value := prompt(reader, stdout, label, defaultValue)
	parsed, err := strconv.Atoi(value)
	if err != nil {
		parsed, _ = strconv.Atoi(defaultValue)
	}
	return parsed
}

func confirm(stdin io.Reader, stdout io.Writer, label string, defaultValue bool) bool {
	reader, ok := stdin.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(stdin)
	}
	suffix := "y/N"
	if defaultValue {
		suffix = "Y/n"
	}
	fmt.Fprintf(stdout, "%s [%s]: ", label, suffix)
	value, _ := reader.ReadString('\n')
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return defaultValue
	}
	return value == "y" || value == "yes"
}

func promptSecret(stdout io.Writer, label string) string {
	fmt.Fprintf(stdout, "%s: ", label)
	if isCharDevice(os.Stdin) {
		if oldState, err := stty("-g"); err == nil {
			_ = sttyNoOutput("-echo")
			reader := bufio.NewReader(os.Stdin)
			value, _ := reader.ReadString('\n')
			_ = sttyNoOutput(oldState)
			fmt.Fprintln(stdout)
			return strings.TrimSpace(value)
		}
	}
	reader := bufio.NewReader(os.Stdin)
	value, _ := reader.ReadString('\n')
	return strings.TrimSpace(value)
}

func isCharDevice(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}

func stty(arg string) (string, error) {
	cmd := exec.Command("stty", arg)
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	return strings.TrimSpace(string(output)), err
}

func sttyNoOutput(arg string) error {
	cmd := exec.Command("stty", arg)
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func valueOrDefault(value, fallback string) string {
	if value == "" || value == "0" {
		return fallback
	}
	return value
}

func printJSON(stdout io.Writer, result SetupResult) {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
}

func printTextResult(stdout io.Writer, result SetupResult, dryRun bool) {
	if dryRun {
		fmt.Fprintln(stdout, "Dry run complete. No Google Cloud resources were changed.")
	} else {
		fmt.Fprintln(stdout, "Setup complete.")
	}
	fmt.Fprintf(stdout, "Service URL: %s\n", result.ServiceURL)
	fmt.Fprintf(stdout, "Result file: %s\n", result.ResultPath)
	for _, step := range result.NextSteps {
		fmt.Fprintf(stdout, "- %s\n", step)
	}
}
