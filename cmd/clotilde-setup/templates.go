package main

import (
	"fmt"
	"strings"
)

func templateConfig(name string) (SetupConfig, error) {
	switch normalizeTemplateName(name) {
	case "minimal":
		return baseTemplate(implementationGeneric, false), nil
	case "full":
		cfg := baseTemplate(implementationGeneric, true)
		enableOptionalAgentSecrets(&cfg)
		return cfg, nil
	case implementationOpenClaw:
		cfg := baseTemplate(implementationOpenClaw, true)
		cfg.ConfigAPI = generatedProvider("clotilde-config")
		return cfg, nil
	case implementationHermes:
		cfg := baseTemplate(implementationHermes, true)
		cfg.Claude = ProviderConfig{
			Enabled: true,
			Secret: SecretConfig{
				SecretName: "clotilde-claude",
				ValueEnv:   "ANTHROPIC_API_KEY",
			},
		}
		cfg.ConfigAPI = generatedProvider("clotilde-config")
		return cfg, nil
	default:
		return SetupConfig{}, fmt.Errorf("unknown template %q; expected minimal, full, openclaw, or hermes", name)
	}
}

func normalizeTemplateName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "minimal"
	}
	return normalizeImplementation(name)
}

func baseTemplate(implementation string, adminEnabled bool) SetupConfig {
	cfg := SetupConfig{
		Implementation: implementation,
		ProjectID:      "your-project-id",
		Region:         "us-central1",
		ServiceName:    "clotilde",
		RepoName:       "clotilde-repo",
		ImageTag:       defaultImageTag,
		LogBufferSize:  1000,
		OpenAI: SecretConfig{
			SecretName: "clotilde-oai",
			ValueEnv:   "OPENAI_API_KEY",
		},
		API: SecretConfig{
			SecretName: "clotilde-auth",
			Generate:   true,
		},
		Admin: AdminConfig{
			Enabled:  adminEnabled,
			Username: "admin",
			Password: SecretConfig{
				SecretName: "clotilde-admin",
				Generate:   true,
			},
		},
	}
	applyDefaults(&cfg)
	return cfg
}

func enableOptionalAgentSecrets(cfg *SetupConfig) {
	cfg.Claude = ProviderConfig{
		Enabled: true,
		Secret: SecretConfig{
			SecretName: "clotilde-claude",
			ValueEnv:   "ANTHROPIC_API_KEY",
		},
	}
	cfg.Perplexity = ProviderConfig{
		Enabled: true,
		Secret: SecretConfig{
			SecretName: "clotilde-perplexity",
			ValueEnv:   "PERPLEXITY_API_KEY",
		},
	}
	cfg.ConfigAPI = generatedProvider("clotilde-config")
}

func generatedProvider(secretName string) ProviderConfig {
	return ProviderConfig{
		Enabled: true,
		Secret: SecretConfig{
			SecretName: secretName,
			Generate:   true,
		},
	}
}
