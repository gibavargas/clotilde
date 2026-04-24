package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func activeProject(ctx context.Context, runner CommandRunner) (string, error) {
	return runner.Run(ctx, CommandSpec{
		Stage: "preflight",
		Name:  "gcloud",
		Args:  []string{"config", "get-value", "project"},
	})
}

func preflight(ctx context.Context, cfg SetupConfig, runner CommandRunner, stderr io.Writer) error {
	if err := runner.LookPath("gcloud"); err != nil {
		return fmt.Errorf("gcloud CLI not found: %w", err)
	}
	if err := runner.LookPath("docker"); err != nil {
		return fmt.Errorf("Docker not found: %w", err)
	}

	project, err := activeProject(ctx, runner)
	if err != nil {
		return err
	}
	if strings.TrimSpace(project) == "" {
		return fmt.Errorf("gcloud has no active project")
	}
	if strings.TrimSpace(project) != cfg.ProjectID {
		_, err = runner.Run(ctx, CommandSpec{
			Stage: "preflight",
			Name:  "gcloud",
			Args:  []string{"config", "set", "project", cfg.ProjectID},
		})
		if err != nil {
			return err
		}
	}

	_, err = runner.Run(ctx, CommandSpec{
		Stage:        "preflight",
		Name:         "gcloud",
		Args:         []string{"billing", "projects", "describe", cfg.ProjectID, "--format=value(billingEnabled)"},
		AllowFailure: true,
	})
	if err != nil {
		fmt.Fprintln(stderr, "Warning: unable to confirm billing status; Cloud Run deployment may fail if billing is disabled.")
	}

	_, err = runner.Run(ctx, CommandSpec{
		Stage: "preflight",
		Name:  "gcloud",
		Args:  []string{"auth", "configure-docker", cfg.Region + "-docker.pkg.dev", "--quiet"},
	})
	return err
}

func provision(ctx context.Context, cfg SetupConfig, runner CommandRunner, stdout io.Writer) error {
	fmt.Fprintln(stdout, "Provisioning Google Cloud resources...")
	if _, err := runner.Run(ctx, CommandSpec{
		Stage: "provision",
		Name:  "gcloud",
		Args: []string{
			"services", "enable",
			"artifactregistry.googleapis.com",
			"secretmanager.googleapis.com",
			"run.googleapis.com",
		},
	}); err != nil {
		return err
	}

	if _, err := runner.Run(ctx, CommandSpec{
		Stage:        "provision",
		Name:         "gcloud",
		Args:         []string{"artifacts", "repositories", "describe", cfg.RepoName, "--location", cfg.Region},
		AllowFailure: true,
	}); err != nil {
		if _, createErr := runner.Run(ctx, CommandSpec{
			Stage: "provision",
			Name:  "gcloud",
			Args: []string{
				"artifacts", "repositories", "create", cfg.RepoName,
				"--repository-format=docker",
				"--location", cfg.Region,
				"--description", "Clotilde Docker images",
				"--quiet",
			},
		}); createErr != nil {
			return createErr
		}
	}

	if err := createOrUpdateSecret(ctx, runner, cfg.OpenAI); err != nil {
		return err
	}
	if err := createOrUpdateSecret(ctx, runner, cfg.API); err != nil {
		return err
	}
	if cfg.Claude.Enabled {
		if err := createOrUpdateSecret(ctx, runner, cfg.Claude.Secret); err != nil {
			return err
		}
	}
	if cfg.Perplexity.Enabled {
		if err := createOrUpdateSecret(ctx, runner, cfg.Perplexity.Secret); err != nil {
			return err
		}
	}
	if cfg.ConfigAPI.Enabled {
		if err := createOrUpdateSecret(ctx, runner, cfg.ConfigAPI.Secret); err != nil {
			return err
		}
	}
	if cfg.Admin.Enabled {
		if err := createOrUpdateSecret(ctx, runner, cfg.Admin.Password); err != nil {
			return err
		}
	}

	projectNumber, err := runner.Run(ctx, CommandSpec{
		Stage: "provision",
		Name:  "gcloud",
		Args:  []string{"projects", "describe", cfg.ProjectID, "--format=value(projectNumber)"},
	})
	if err != nil {
		return err
	}
	serviceAccount := strings.TrimSpace(projectNumber) + "-compute@developer.gserviceaccount.com"
	for _, secretName := range orderedSecretNames(cfg) {
		if err := grantSecretAccess(ctx, runner, secretName, serviceAccount); err != nil {
			return err
		}
	}
	return nil
}

func createOrUpdateSecret(ctx context.Context, runner CommandRunner, secret SecretConfig) error {
	if secret.UseExistingSecret {
		return nil
	}
	value, err := resolveSecretValue(secret)
	if err != nil {
		return err
	}
	if value == "" {
		return fmt.Errorf("no value available for secret %s", secret.SecretName)
	}

	if _, err := runner.Run(ctx, CommandSpec{
		Stage:        "provision",
		Name:         "gcloud",
		Args:         []string{"secrets", "describe", secret.SecretName},
		AllowFailure: true,
	}); err != nil {
		_, err = runner.Run(ctx, CommandSpec{
			Stage:          "provision",
			Name:           "gcloud",
			Args:           []string{"secrets", "create", secret.SecretName, "--data-file=-", "--replication-policy=automatic", "--quiet"},
			Stdin:          value,
			SensitiveStdin: true,
		})
		return err
	}

	_, err = runner.Run(ctx, CommandSpec{
		Stage:          "provision",
		Name:           "gcloud",
		Args:           []string{"secrets", "versions", "add", secret.SecretName, "--data-file=-"},
		Stdin:          value,
		SensitiveStdin: true,
	})
	return err
}

func grantSecretAccess(ctx context.Context, runner CommandRunner, secretName, serviceAccount string) error {
	_, err := runner.Run(ctx, CommandSpec{
		Stage: "provision",
		Name:  "gcloud",
		Args: []string{
			"secrets", "add-iam-policy-binding", secretName,
			"--member", "serviceAccount:" + serviceAccount,
			"--role", "roles/secretmanager.secretAccessor",
			"--quiet",
		},
	})
	return err
}

func deploy(ctx context.Context, cfg SetupConfig, opts Options, runner CommandRunner, stdout io.Writer) (string, error) {
	imageName := fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s:%s", cfg.Region, cfg.ProjectID, cfg.RepoName, cfg.ServiceName, cfg.ImageTag)
	if !opts.SkipBuild {
		fmt.Fprintln(stdout, "Building Docker image...")
		if _, err := runner.Run(ctx, CommandSpec{
			Stage: "deploy",
			Name:  "docker",
			Args:  []string{"build", "-t", imageName, "."},
		}); err != nil {
			return "", err
		}

		fmt.Fprintln(stdout, "Pushing Docker image...")
		if _, err := runner.Run(ctx, CommandSpec{
			Stage: "deploy",
			Name:  "docker",
			Args:  []string{"push", imageName},
		}); err != nil {
			return "", err
		}
	}

	fmt.Fprintln(stdout, "Deploying to Cloud Run...")
	args := []string{
		"run", "deploy", cfg.ServiceName,
		"--image", imageName,
		"--region", cfg.Region,
		"--platform", "managed",
		"--allow-unauthenticated",
		"--memory", "256Mi",
		"--cpu", "1",
		"--min-instances", "0",
		"--max-instances", "10",
		"--timeout", "60",
		"--set-env-vars", buildEnvVars(cfg, opts.DryRun),
		"--set-secrets", buildSecretVars(cfg),
		"--quiet",
	}
	if _, err := runner.Run(ctx, CommandSpec{Stage: "deploy", Name: "gcloud", Args: args}); err != nil {
		return "", err
	}
	return describeServiceURL(ctx, cfg, runner)
}

func buildEnvVars(cfg SetupConfig, dryRun bool) string {
	ipSalt, err := randomHex(32)
	if err != nil {
		ipSalt = "setup-generated-salt"
	}
	if dryRun {
		ipSalt = "dry-run-ip-hash-salt"
	}
	env := []string{
		"GOOGLE_CLOUD_PROJECT=" + cfg.ProjectID,
		fmt.Sprintf("LOG_BUFFER_SIZE=%d", cfg.LogBufferSize),
		"IP_HASH_SALT=" + ipSalt,
	}
	if cfg.Admin.Enabled {
		env = append(env, "ADMIN_USER="+cfg.Admin.Username)
	}
	return strings.Join(env, ",")
}

func buildSecretVars(cfg SetupConfig) string {
	secrets := []string{
		"OPENAI_KEY_SECRET_NAME=" + cfg.OpenAI.SecretName + ":latest",
		"API_KEY_SECRET_NAME=" + cfg.API.SecretName + ":latest",
	}
	if cfg.Admin.Enabled {
		secrets = append(secrets, "ADMIN_PASSWORD="+cfg.Admin.Password.SecretName+":latest")
	}
	if cfg.ConfigAPI.Enabled {
		secrets = append(secrets, "CONFIG_API_KEY="+cfg.ConfigAPI.Secret.SecretName+":latest")
	}
	if cfg.Perplexity.Enabled {
		secrets = append(secrets, "PERPLEXITY_KEY_SECRET_NAME="+cfg.Perplexity.Secret.SecretName+":latest")
	}
	if cfg.Claude.Enabled {
		secrets = append(secrets, "CLAUDE_KEY_SECRET_NAME="+cfg.Claude.Secret.SecretName+":latest")
	}
	return strings.Join(secrets, ",")
}

func describeServiceURL(ctx context.Context, cfg SetupConfig, runner CommandRunner) (string, error) {
	output, err := runner.Run(ctx, CommandSpec{
		Stage: "verify",
		Name:  "gcloud",
		Args:  []string{"run", "services", "describe", cfg.ServiceName, "--region", cfg.Region, "--format=value(status.url)"},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func verify(ctx context.Context, serviceURL string, cfg SetupConfig, opts Options, stdout io.Writer) error {
	if opts.DryRun {
		fmt.Fprintln(stdout, "Skipping network verification in dry-run mode.")
		return nil
	}
	if serviceURL == "" {
		return fmt.Errorf("service URL is empty")
	}
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(serviceURL, "/")+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("/health returned %d", resp.StatusCode)
	}

	adminReq, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(serviceURL, "/")+"/admin/", nil)
	if err != nil {
		return err
	}
	adminResp, err := client.Do(adminReq)
	if err != nil {
		return err
	}
	_ = adminResp.Body.Close()
	expected := http.StatusServiceUnavailable
	if cfg.Admin.Enabled {
		expected = http.StatusUnauthorized
	}
	if adminResp.StatusCode != expected {
		return fmt.Errorf("/admin/ returned %d, expected %d", adminResp.StatusCode, expected)
	}
	return nil
}

func nextSteps(serviceURL string, cfg SetupConfig) []string {
	url := strings.TrimRight(serviceURL, "/")
	steps := []string{
		"Retrieve your service API key with: gcloud secrets versions access latest --secret=" + cfg.API.SecretName,
		"Test chat with: curl -X POST " + url + "/chat -H 'Content-Type: application/json' -H 'X-API-Key: YOUR_API_KEY' -d '{\"message\":\"teste\"}'",
	}
	if cfg.Admin.Enabled {
		steps = append(steps, "Open the admin dashboard at "+url+"/admin/ and sign in as "+cfg.Admin.Username+".")
	}
	return steps
}

func writeResult(path string, result SetupResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
