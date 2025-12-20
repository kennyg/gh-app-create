# gh-app-create

A [GitHub CLI](https://cli.github.com/) extension for creating GitHub Apps programmatically using the [manifest flow](https://docs.github.com/en/apps/sharing-github-apps/registering-a-github-app-from-a-manifest).

## Why?

Creating a GitHub App through the web UI requires clicking through 15+ form fields and manually configuring permissions. This extension lets you:

- **Create apps from presets** - Common configurations like CI bots, release automation, etc.
- **Create apps from YAML manifests** - Version-controlled, reproducible app configs
- **Works with GitHub Enterprise Server** - Just pass `--hostname`

## Installation

```bash
gh extension install kennyg/gh-app-create
```

## Usage

### Create from a preset

```bash
# List available presets
gh app-create presets

# Create an app for your organization
gh app-create create --preset opencode --org my-org

# Create for your personal account
gh app-create create --preset ci-bot
```

### Create from a custom manifest

```bash
# Export a preset to customize
gh app-create export --preset opencode --output my-app.yaml

# Edit the manifest...
vim my-app.yaml

# Create from your custom manifest
gh app-create create --manifest my-app.yaml --org my-org
```

### GitHub Enterprise Server

```bash
gh app-create create --preset ci-bot --org my-org --hostname github.mycompany.com
```

### Output formats

```bash
# YAML (default)
gh app-create create --preset ci-bot --format yaml

# JSON
gh app-create create --preset ci-bot --format json

# Environment variables
gh app-create create --preset ci-bot --format env --output .env

# GitHub Actions secrets format
gh app-create create --preset ci-bot --format github-actions
```

## Available Presets

| Preset | Description | Permissions |
|--------|-------------|-------------|
| `opencode` | AI coding agent for issues/PRs | contents:write, pull_requests:write, issues:write |
| `renovate` | Self-hosted Renovate bot | contents:write, pull_requests:write, issues:write, checks:write, statuses:write, workflows:write, administration:read, dependabot_alerts:read, members:read |
| `ci-bot` | CI/CD automation | contents:write, pull_requests:write, checks:write, statuses:write |
| `release-bot` | Release automation | contents:write, pull_requests:write, issues:write |
| `readonly` | Read-only analysis tools | contents:read, metadata:read |
| `actions-token` | Token generation for Actions | contents:write, pull_requests:write, issues:write, actions:write |

## Manifest Format

```yaml
name: my-app
url: https://example.com
description: My custom GitHub App
public: false
hook_attributes:
  active: false
default_permissions:
  contents: write
  pull_requests: write
  issues: read
  metadata: read
default_events:
  - push
  - pull_request
  - issue_comment
```

### Permission values

- `read` - Read-only access
- `write` - Read and write access

### Available permissions

See [GitHub's full list](https://docs.github.com/en/rest/overview/permissions-required-for-github-apps).

Common ones:
- `contents` - Repository contents (code, commits)
- `pull_requests` - Pull requests
- `issues` - Issues
- `checks` - Check runs and suites
- `statuses` - Commit statuses
- `actions` - GitHub Actions workflows
- `metadata` - Basic repo info (always read-only)

## After Creation

Once your app is created:

1. **Install the app** on repositories using the link provided
2. **Generate tokens** using the [gh-token extension](https://github.com/Link-/gh-token):

```bash
# Install gh-token
gh extension install Link-/gh-token

# Generate an installation access token
gh token generate --app-id <APP_ID> --key ./private-key.pem
```

## In GitHub Actions

```yaml
- name: Generate GitHub App Token
  id: app-token
  uses: actions/create-github-app-token@v2
  with:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}

- name: Use the token
  env:
    GH_TOKEN: ${{ steps.app-token.outputs.token }}
  run: |
    gh pr create --fill
```

## How it works

This extension uses the [GitHub App Manifest flow](https://docs.github.com/en/apps/sharing-github-apps/registering-a-github-app-from-a-manifest):

1. Opens your browser with a pre-filled manifest
2. You click "Create GitHub App" on GitHub
3. GitHub redirects back to a local callback server
4. The extension exchanges the temporary code for credentials
5. Outputs the App ID, private key, and other secrets

## Security

- Credentials are output to stdout or a file you specify
- Private keys should be stored securely (e.g., in a secret manager)
- The local callback server only runs briefly during creation

## License

MIT

## See Also

- [gh-token](https://github.com/Link-/gh-token) - Generate installation tokens for existing apps
- [actions/create-github-app-token](https://github.com/actions/create-github-app-token) - Generate tokens in GitHub Actions
