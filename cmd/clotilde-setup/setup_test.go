package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func validTestConfig() SetupConfig {
	return SetupConfig{
		ProjectID:     "test-project",
		Region:        "us-central1",
		ServiceName:   "clotilde",
		RepoName:      "clotilde-repo",
		ImageTag:      "latest",
		LogBufferSize: 1000,
		OpenAI: SecretConfig{
			SecretName: "clotilde-oai-test",
			Value:      "test-openai-secret",
		},
		API: SecretConfig{
			SecretName: "clotilde-auth-test",
			Generate:   true,
		},
		Admin: AdminConfig{
			Enabled:  true,
			Username: "admin",
			Password: SecretConfig{
				SecretName: "clotilde-admin-test",
				Value:      "admin-password",
			},
		},
		Claude: ProviderConfig{
			Enabled: true,
			Secret: SecretConfig{
				SecretName: "clotilde-claude-test",
				Value:      "test-claude-secret",
			},
		},
		Perplexity: ProviderConfig{
			Enabled: true,
			Secret: SecretConfig{
				SecretName: "clotilde-perplexity-test",
				Value:      "test-perplexity-secret",
			},
		},
		ConfigAPI: ProviderConfig{
			Enabled: true,
			Secret: SecretConfig{
				SecretName: "clotilde-config-test",
				Value:      "config-key",
			},
		},
	}
}

func TestValidateConfigRequiredFields(t *testing.T) {
	cfg := validTestConfig()
	cfg.ProjectID = ""

	err := validateConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "project_id is required") {
		t.Fatalf("expected missing project validation error, got %v", err)
	}
}

func TestValidateConfigInvalidSecretName(t *testing.T) {
	cfg := validTestConfig()
	cfg.OpenAI.SecretName = "invalid secret name"

	err := validateConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "openai.secret_name contains invalid characters") {
		t.Fatalf("expected invalid secret validation error, got %v", err)
	}
}

func TestValidateConfigAdminRequiresUsernameAndPasswordSource(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Username = ""
	cfg.Admin.Password = SecretConfig{SecretName: "clotilde-admin-test"}

	err := validateConfig(cfg)
	if err == nil {
		t.Fatal("expected admin validation error")
	}
	if !strings.Contains(err.Error(), "admin.username is required") {
		t.Fatalf("expected admin username error, got %v", err)
	}
	if !strings.Contains(err.Error(), "admin.password must set use_existing_secret, generate, value, or value_env") {
		t.Fatalf("expected admin password source error, got %v", err)
	}
}

func TestValidateConfigOptionalProvidersCanBeDisabled(t *testing.T) {
	cfg := validTestConfig()
	cfg.Claude = ProviderConfig{}
	cfg.Perplexity = ProviderConfig{}
	cfg.ConfigAPI = ProviderConfig{}

	if err := validateConfig(cfg); err != nil {
		t.Fatalf("disabled optional providers should not require secrets: %v", err)
	}
}

func TestDryRunCommandPlanningAndRedaction(t *testing.T) {
	cfg := validTestConfig()
	var stdout, stderr bytes.Buffer
	runner := &DryRunRunner{ProjectID: cfg.ProjectID, Region: cfg.Region, ServiceName: cfg.ServiceName}
	opts := Options{
		NonInteractive: true,
		Output:         "json",
		Yes:            true,
		DryRun:         true,
		ResultPath:     ".clotilde/test-result.json",
	}

	result, code := runSetup(context.Background(), cfg, opts, runner, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("runSetup returned code %d error %s stderr %s", code, result.Error, stderr.String())
	}
	if !result.OK {
		t.Fatal("expected OK result")
	}

	commands := runner.Commands()
	requireCommandOrder(t, commands, "preflight:gcloud:config get-value project", "provision:gcloud:services enable", "provision:gcloud:artifacts repositories describe", "provision:gcloud:artifacts repositories create", "deploy:docker:build", "deploy:docker:push", "deploy:gcloud:run deploy", "verify:gcloud:run services describe")

	deployArgs := findCommandArgs(commands, "deploy", "gcloud", "run", "deploy")
	if deployArgs == nil {
		t.Fatal("expected gcloud run deploy command")
	}
	deployJoined := strings.Join(deployArgs, " ")
	for _, expected := range []string{
		"OPENAI_KEY_SECRET_NAME=clotilde-oai-test:latest",
		"API_KEY_SECRET_NAME=clotilde-auth-test:latest",
		"ADMIN_PASSWORD=clotilde-admin-test:latest",
		"CONFIG_API_KEY=clotilde-config-test:latest",
		"PERPLEXITY_KEY_SECRET_NAME=clotilde-perplexity-test:latest",
		"CLAUDE_KEY_SECRET_NAME=clotilde-claude-test:latest",
		"ADMIN_USER=admin",
	} {
		if !strings.Contains(deployJoined, expected) {
			t.Fatalf("deploy command missing %q in %s", expected, deployJoined)
		}
	}

	output := stdout.String()
	for _, secretValue := range []string{"test-openai-secret", "admin-password", "test-claude-secret", "test-perplexity-secret", "config-key"} {
		if strings.Contains(output, secretValue) {
			t.Fatalf("stdout leaked secret value %q", secretValue)
		}
	}
}

func TestDryRunCLIProducesJSON(t *testing.T) {
	cfg := validTestConfig()
	cfg.Claude = ProviderConfig{}
	cfg.Perplexity = ProviderConfig{}
	cfg.ConfigAPI = ProviderConfig{}

	tmp := t.TempDir()
	configPath := tmp + "/setup.json"
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", ".", "--non-interactive", "--config", configPath, "--output", "json", "--yes", "--dry-run")
	cmd.Dir = "."
	var cmdStderr bytes.Buffer
	cmd.Stderr = &cmdStderr
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go run dry-run failed: %v\nstdout:\n%s\nstderr:\n%s", err, string(output), cmdStderr.String())
	}

	var result SetupResult
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("expected JSON output, got error %v\n%s", err, string(output))
	}
	if !result.OK {
		t.Fatalf("expected OK result: %+v", result)
	}
	if result.ServiceURL == "" {
		t.Fatal("expected service URL in dry-run output")
	}
	if len(result.Commands) == 0 {
		t.Fatal("expected command list in dry-run output")
	}
}

func requireCommandOrder(t *testing.T, commands []LoggedCommand, expected ...string) {
	t.Helper()
	next := 0
	for _, cmd := range commands {
		if next >= len(expected) {
			return
		}
		if commandMatches(cmd, expected[next]) {
			next++
		}
	}
	if next != len(expected) {
		t.Fatalf("missing expected command order item %q in commands %+v", expected[next], commands)
	}
}

func commandMatches(cmd LoggedCommand, pattern string) bool {
	parts := strings.SplitN(pattern, ":", 3)
	if len(parts) != 3 {
		return false
	}
	if cmd.Stage != parts[0] || cmd.Name != parts[1] {
		return false
	}
	return strings.HasPrefix(strings.Join(cmd.Args, " "), parts[2])
}

func findCommandArgs(commands []LoggedCommand, stage, name string, argPrefix ...string) []string {
	prefix := strings.Join(argPrefix, " ")
	for _, cmd := range commands {
		if cmd.Stage == stage && cmd.Name == name && strings.HasPrefix(strings.Join(cmd.Args, " "), prefix) {
			return cmd.Args
		}
	}
	return nil
}
