package main

import "time"

// SetupConfig is the machine-readable input contract for the deployment wizard.
type SetupConfig struct {
	ProjectID     string         `json:"project_id"`
	Region        string         `json:"region"`
	ServiceName   string         `json:"service_name"`
	RepoName      string         `json:"repo_name"`
	ImageTag      string         `json:"image_tag,omitempty"`
	LogBufferSize int            `json:"log_buffer_size,omitempty"`
	OpenAI        SecretConfig   `json:"openai"`
	API           SecretConfig   `json:"api"`
	Claude        ProviderConfig `json:"claude,omitempty"`
	Perplexity    ProviderConfig `json:"perplexity,omitempty"`
	ConfigAPI     ProviderConfig `json:"config_api,omitempty"`
	Admin         AdminConfig    `json:"admin,omitempty"`
}

// SecretConfig describes a Secret Manager secret and how its value should be sourced.
type SecretConfig struct {
	SecretName        string `json:"secret_name"`
	Value             string `json:"value,omitempty"`
	ValueEnv          string `json:"value_env,omitempty"`
	UseExistingSecret bool   `json:"use_existing_secret,omitempty"`
	Generate          bool   `json:"generate,omitempty"`
}

// ProviderConfig describes an optional API provider.
type ProviderConfig struct {
	Enabled bool         `json:"enabled"`
	Secret  SecretConfig `json:"secret"`
}

// AdminConfig describes optional admin dashboard provisioning.
type AdminConfig struct {
	Enabled  bool         `json:"enabled"`
	Username string       `json:"username,omitempty"`
	Password SecretConfig `json:"password"`
}

// SetupResult is the machine-readable output contract for OpenClaw-style agents.
type SetupResult struct {
	OK           bool              `json:"ok"`
	ServiceURL   string            `json:"service_url,omitempty"`
	Secrets      map[string]string `json:"secrets,omitempty"`
	AdminEnabled bool              `json:"admin_enabled"`
	NextSteps    []string          `json:"next_steps,omitempty"`
	ResultPath   string            `json:"result_path,omitempty"`
	Commands     []LoggedCommand   `json:"commands,omitempty"`
	Error        string            `json:"error,omitempty"`
	StartedAt    time.Time         `json:"started_at,omitempty"`
	FinishedAt   time.Time         `json:"finished_at,omitempty"`
}

// LoggedCommand is a sanitized command record used by dry-run and tests.
type LoggedCommand struct {
	Stage string   `json:"stage"`
	Name  string   `json:"name"`
	Args  []string `json:"args"`
}

// Options captures CLI behavior flags.
type Options struct {
	NonInteractive bool
	ConfigPath     string
	Output         string
	Yes            bool
	DryRun         bool
	SkipBuild      bool
	SkipVerify     bool
	ResultPath     string
}
