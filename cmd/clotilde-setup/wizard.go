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
	applyDefaults(&cfg)
	result := SetupResult{
		OK:             false,
		Implementation: cfg.Implementation,
		Secrets:        map[string]string{},
		AdminEnabled:   cfg.Admin.Enabled,
		StartedAt:      startedAt,
	}
	progressOut := stdout
	if opts.Output == "json" {
		progressOut = stderr
	}

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
	result.Implementation = cfg.Implementation
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
	var stdinFile *os.File
	if file, ok := stdin.(*os.File); ok {
		stdinFile = file
	}

	fmt.Fprintln(stdout, "Clotilde setup wizard")
	fmt.Fprintln(stdout, "Press Enter to accept defaults. Secret values are never written to the setup result.")
	cfg.Implementation = promptChoice(reader, stdout, "Implementation profile (generic/openclaw/hermes)", cfg.Implementation, []string{implementationGeneric, implementationOpenClaw, implementationHermes})
	quick := confirm(reader, stdout, "Use quick Cloud Run defaults?", true)

	active, _ := activeProject(ctx, runner)
	active = strings.TrimSpace(active)
	if cfg.ProjectID == "" {
		cfg.ProjectID = prompt(reader, stdout, "Google Cloud project ID", active)
	}
	cfg.Region = prompt(reader, stdout, "Region", valueOrDefault(cfg.Region, "us-central1"))
	if quick {
		fmt.Fprintf(stdout, "Using service %q, repo %q, image tag %q, and log buffer %d.\n", cfg.ServiceName, cfg.RepoName, cfg.ImageTag, cfg.LogBufferSize)
	} else {
		cfg.ServiceName = prompt(reader, stdout, "Cloud Run service name", valueOrDefault(cfg.ServiceName, "clotilde"))
		cfg.RepoName = prompt(reader, stdout, "Artifact Registry repo name", valueOrDefault(cfg.RepoName, "clotilde-repo"))
		cfg.ImageTag = prompt(reader, stdout, "Image tag", valueOrDefault(cfg.ImageTag, defaultImageTag))
		cfg.LogBufferSize = promptInt(reader, stdout, "Log buffer size", valueOrDefault(strconv.Itoa(cfg.LogBufferSize), "1000"))
	}

	cfg.API = collectSecretSource(reader, stdinFile, stdout, cfg.API, "Service API key", defaultSecretName("clotilde-auth"), []string{"CLOTILDE_API_KEY", "API_KEY_SECRET_NAME"}, true, true)

	cfg.Admin.Enabled = confirm(reader, stdout, "Enable admin dashboard?", cfg.Admin.Enabled || quick)
	if cfg.Admin.Enabled {
		cfg.Admin.Username = prompt(reader, stdout, "Admin username", valueOrDefault(cfg.Admin.Username, "admin"))
		cfg.Admin.Password = collectSecretSource(reader, stdinFile, stdout, cfg.Admin.Password, "Admin password", defaultSecretName("clotilde-admin"), []string{"CLOTILDE_ADMIN_PASSWORD", "ADMIN_PASSWORD"}, true, true)
	}

	cfg.Claude.Enabled = confirm(reader, stdout, "Enable Claude Haiku direct API?", cfg.Claude.Enabled || cfg.Implementation == implementationGeneric || cfg.Implementation == implementationHermes)
	if cfg.Claude.Enabled {
		cfg.Claude.Secret = collectSecretSource(reader, stdinFile, stdout, cfg.Claude.Secret, "Claude API key", defaultSecretName("clotilde-claude"), []string{"ANTHROPIC_API_KEY", "CLAUDE_API_KEY", "CLAUDE_KEY_SECRET_NAME"}, false, false)
	}

	cfg.OpenRouter.Enabled = confirm(reader, stdout, "Enable OpenRouter fallback?", cfg.OpenRouter.Enabled)
	if cfg.OpenRouter.Enabled {
		cfg.OpenRouter.Secret = collectSecretSource(reader, stdinFile, stdout, cfg.OpenRouter.Secret, "OpenRouter API key", defaultSecretName("clotilde-openrouter"), []string{"OPENROUTER_API_KEY", "OPENROUTER_KEY_SECRET_NAME"}, false, false)
	}

	if isSecretConfigured(cfg.OpenAI) || confirm(reader, stdout, "Enable OpenAI Responses fallback?", cfg.OpenAI.SecretName != "" || cfg.Implementation == implementationOpenClaw) {
		cfg.OpenAI = collectSecretSource(reader, stdinFile, stdout, cfg.OpenAI, "OpenAI API key", defaultSecretName("clotilde-oai"), []string{"OPENAI_API_KEY", "OPENAI_KEY_SECRET_NAME"}, false, false)
	}

	cfg.Perplexity.Enabled = confirm(reader, stdout, "Enable Perplexity search?", cfg.Perplexity.Enabled)
	if cfg.Perplexity.Enabled {
		cfg.Perplexity.Secret = collectSecretSource(reader, stdinFile, stdout, cfg.Perplexity.Secret, "Perplexity API key", defaultSecretName("clotilde-perplexity"), []string{"PERPLEXITY_API_KEY", "PERPLEXITY_KEY_SECRET_NAME"}, false, false)
	}

	cfg.ConfigAPI.Enabled = confirm(reader, stdout, "Enable dedicated config API key?", cfg.ConfigAPI.Enabled || cfg.Implementation == implementationOpenClaw || cfg.Implementation == implementationHermes)
	if cfg.ConfigAPI.Enabled {
		cfg.ConfigAPI.Secret = collectSecretSource(reader, stdinFile, stdout, cfg.ConfigAPI.Secret, "Config API key", defaultSecretName("clotilde-config"), []string{"CLOTILDE_CONFIG_API_KEY", "CONFIG_API_KEY"}, true, true)
	}

	return cfg, nil
}

func promptChoice(reader *bufio.Reader, stdout io.Writer, label, defaultValue string, allowed []string) string {
	allowedSet := map[string]bool{}
	for _, value := range allowed {
		allowedSet[value] = true
	}
	for {
		value := normalizeImplementation(prompt(reader, stdout, label, defaultValue))
		if allowedSet[value] {
			return value
		}
		fmt.Fprintf(stdout, "Choose one of: %s\n", strings.Join(allowed, ", "))
	}
}

func collectSecretSource(reader *bufio.Reader, stdinFile *os.File, stdout io.Writer, secret SecretConfig, label, defaultSecret string, envCandidates []string, allowGenerate, defaultGenerate bool) SecretConfig {
	if secret.SecretName == "" {
		secret.SecretName = prompt(reader, stdout, label+" secret name", defaultSecret)
	}
	if hasSecretSource(secret) {
		return secret
	}

	if allowGenerate && confirm(reader, stdout, "Generate "+strings.ToLower(label)+" automatically?", defaultGenerate) {
		secret.Generate = true
		return secret
	}

	if envName := firstSetEnv(envCandidates); envName != "" {
		if confirm(reader, stdout, "Use "+envName+" from your environment for "+label+"?", true) {
			secret.ValueEnv = envName
			return secret
		}
	}

	if confirm(reader, stdout, "Use an existing Secret Manager secret for "+label+"?", false) {
		secret.UseExistingSecret = true
		return secret
	}

	for {
		secret.Value = promptSecret(reader, stdinFile, stdout, label)
		if strings.TrimSpace(secret.Value) != "" {
			return secret
		}
		fmt.Fprintf(stdout, "%s is required unless you use an existing Secret Manager secret.\n", label)
		if confirm(reader, stdout, "Use an existing Secret Manager secret for "+label+"?", false) {
			secret.UseExistingSecret = true
			return secret
		}
	}
}

func firstSetEnv(names []string) string {
	for _, name := range names {
		if os.Getenv(name) != "" {
			return name
		}
	}
	return ""
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

func promptSecret(reader *bufio.Reader, stdinFile *os.File, stdout io.Writer, label string) string {
	fmt.Fprintf(stdout, "%s: ", label)
	if stdinFile != nil && isCharDevice(stdinFile) {
		if oldState, err := stty(stdinFile, "-g"); err == nil {
			_ = sttyNoOutput(stdinFile, "-echo")
			value, _ := reader.ReadString('\n')
			_ = sttyNoOutput(stdinFile, oldState)
			fmt.Fprintln(stdout)
			return strings.TrimSpace(value)
		}
	}
	value, _ := reader.ReadString('\n')
	return strings.TrimSpace(value)
}

func isCharDevice(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}

func stty(stdin *os.File, arg string) (string, error) {
	cmd := exec.Command("stty", arg)
	cmd.Stdin = stdin
	output, err := cmd.Output()
	return strings.TrimSpace(string(output)), err
}

func sttyNoOutput(stdin *os.File, arg string) error {
	cmd := exec.Command("stty", arg)
	cmd.Stdin = stdin
	return cmd.Run()
}

func valueOrDefault(value, fallback string) string {
	if value == "" || value == "0" {
		return fallback
	}
	return value
}

func printJSON(stdout io.Writer, result any) {
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
