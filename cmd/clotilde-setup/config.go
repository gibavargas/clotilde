package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var secretNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,255}$`)

func loadConfig(path string) (SetupConfig, error) {
	var cfg SetupConfig
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func applyDefaults(cfg *SetupConfig) {
	if cfg.Region == "" {
		cfg.Region = "us-central1"
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "clotilde"
	}
	if cfg.RepoName == "" {
		cfg.RepoName = "clotilde-repo"
	}
	if cfg.ImageTag == "" {
		cfg.ImageTag = "latest"
	}
	if cfg.LogBufferSize == 0 {
		cfg.LogBufferSize = 1000
	}
}

func validateConfig(cfg SetupConfig) error {
	var problems []string
	if strings.TrimSpace(cfg.ProjectID) == "" {
		problems = append(problems, "project_id is required")
	}
	if strings.TrimSpace(cfg.Region) == "" {
		problems = append(problems, "region is required")
	}
	if strings.TrimSpace(cfg.ServiceName) == "" {
		problems = append(problems, "service_name is required")
	}
	if strings.TrimSpace(cfg.RepoName) == "" {
		problems = append(problems, "repo_name is required")
	}
	if cfg.LogBufferSize < 0 {
		problems = append(problems, "log_buffer_size must be non-negative")
	}

	problems = append(problems, validateSecret("openai", cfg.OpenAI, true)...)
	problems = append(problems, validateSecret("api", cfg.API, true)...)
	problems = append(problems, validateProvider("claude", cfg.Claude)...)
	problems = append(problems, validateProvider("perplexity", cfg.Perplexity)...)
	problems = append(problems, validateProvider("config_api", cfg.ConfigAPI)...)

	if cfg.Admin.Enabled {
		if strings.TrimSpace(cfg.Admin.Username) == "" {
			problems = append(problems, "admin.username is required when admin.enabled is true")
		}
		problems = append(problems, validateSecret("admin.password", cfg.Admin.Password, true)...)
	}

	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func validateProvider(name string, provider ProviderConfig) []string {
	if !provider.Enabled {
		return nil
	}
	return validateSecret(name+".secret", provider.Secret, true)
}

func validateSecret(path string, secret SecretConfig, requireValueSource bool) []string {
	var problems []string
	if strings.TrimSpace(secret.SecretName) == "" {
		problems = append(problems, path+".secret_name is required")
	} else if !secretNamePattern.MatchString(secret.SecretName) {
		problems = append(problems, path+".secret_name contains invalid characters")
	}

	if requireValueSource && !secret.UseExistingSecret && !secret.Generate && secret.Value == "" && secret.ValueEnv == "" {
		problems = append(problems, path+" must set use_existing_secret, generate, value, or value_env")
	}
	return problems
}

func resolveSecretValue(secret SecretConfig) (string, error) {
	if secret.UseExistingSecret {
		return "", nil
	}
	if secret.Generate {
		return randomHex(32)
	}
	if secret.ValueEnv != "" {
		value := os.Getenv(secret.ValueEnv)
		if value == "" {
			return "", fmt.Errorf("environment variable %s is empty", secret.ValueEnv)
		}
		return value, nil
	}
	if secret.Value != "" {
		return secret.Value, nil
	}
	return "", nil
}

func randomHex(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func secretInventory(cfg SetupConfig) map[string]string {
	secrets := map[string]string{
		"openai": cfg.OpenAI.SecretName,
		"api":    cfg.API.SecretName,
	}
	if cfg.Claude.Enabled {
		secrets["claude"] = cfg.Claude.Secret.SecretName
	}
	if cfg.Perplexity.Enabled {
		secrets["perplexity"] = cfg.Perplexity.Secret.SecretName
	}
	if cfg.ConfigAPI.Enabled {
		secrets["config_api"] = cfg.ConfigAPI.Secret.SecretName
	}
	if cfg.Admin.Enabled {
		secrets["admin"] = cfg.Admin.Password.SecretName
	}
	return secrets
}

func orderedSecretNames(cfg SetupConfig) []string {
	secrets := []string{cfg.OpenAI.SecretName, cfg.API.SecretName}
	if cfg.Admin.Enabled {
		secrets = append(secrets, cfg.Admin.Password.SecretName)
	}
	if cfg.ConfigAPI.Enabled {
		secrets = append(secrets, cfg.ConfigAPI.Secret.SecretName)
	}
	if cfg.Perplexity.Enabled {
		secrets = append(secrets, cfg.Perplexity.Secret.SecretName)
	}
	if cfg.Claude.Enabled {
		secrets = append(secrets, cfg.Claude.Secret.SecretName)
	}
	return secrets
}

func defaultSecretName(prefix string) string {
	suffix, err := randomHex(4)
	if err != nil {
		suffix = strconv.FormatInt(int64(os.Getpid()), 10)
	}
	return prefix + "-" + suffix
}
