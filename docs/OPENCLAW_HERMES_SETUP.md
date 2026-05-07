# OpenClaw and Hermes Setup

This project ships a small Go setup wizard for agent-driven deployments. It can be used interactively by a person, or non-interactively by OpenClaw and Hermes implementations that need a predictable JSON contract.

The profile names are:

- `generic`: plain Clotilde deployment centered on Claude Haiku.
- `openclaw`: enables OpenAI fallback plus a generated `config_api` handoff secret and writes a handoff result for OpenClaw agents.
- `hermes`: enables Claude/Anthropic support by default, enables a generated `config_api` handoff secret, and writes the same handoff result for Hermes implementations.

The runtime `/api/config` endpoint still uses the main service `X-API-Key`. The `config_api` secret is included in the setup result for agent workflows that need a separate generated value in their own handoff contract.

## Interactive Wizard

```bash
go run ./cmd/clotilde-setup --wizard --implementation openclaw
```

```bash
go run ./cmd/clotilde-setup --wizard --implementation hermes
```

The wizard starts with quick Cloud Run defaults. Choose the advanced path only if you need custom service, repository, image tag, or log buffer settings.

## JSON Templates

Generate a starter config for OpenClaw:

```bash
go run ./cmd/clotilde-setup --template openclaw > setup.openclaw.json
```

Generate a starter config for Hermes:

```bash
go run ./cmd/clotilde-setup --template hermes > setup.hermes.json
```

The generated templates reference environment variables instead of embedding secret values. Set the relevant variables before running setup:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENROUTER_API_KEY="sk-or-v1-..."
```

`ANTHROPIC_API_KEY` is the recommended primary path. `OPENAI_API_KEY` and `OPENROUTER_API_KEY` are optional fallback/provider paths depending on the template you choose.

Then run:

```bash
go run ./cmd/clotilde-setup \
  --non-interactive \
  --config setup.hermes.json \
  --output json \
  --yes
```

Use `--dry-run` first to inspect the exact Google Cloud and Docker commands without changing resources.

## Handoff Result

Successful setup writes `.clotilde/setup-result.json`. The file includes:

- `implementation`
- `service_url`
- secret names, never secret values
- the generated `config_api` secret name when that section is enabled
- next-step commands for generated API keys
- the sanitized command inventory used during setup

This mirrors the OpenClaw expectation that setup flows can run as a guided wizard or in non-interactive automation, while keeping Hermes-friendly provider defaults available from a single command.

References:

- [OpenClaw setup command](https://docs.openclaw.ai/es/cli/setup)
- [NemoClaw Hermes quickstart](https://docs.nvidia.com/nemoclaw/latest/get-started/quickstart-hermes.html)
