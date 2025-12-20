# Setting Up OpenCode Agent with Your Own GitHub App

This guide explains how to set up the [OpenCode GitHub agent](https://github.com/sst/opencode/tree/dev/github) using a self-managed GitHub App created with `gh-app-create`. This approach works with both GitHub.com and GitHub Enterprise Server (GHES).

## Why Use Your Own GitHub App?

The public `opencode-agent` app uses an OIDC token exchange flow through `api.opencode.ai`. Using your own app:

- **Works with GitHub Enterprise Server** - No dependency on external OIDC endpoints
- **Full control** - You own the app credentials and can revoke access anytime
- **Privacy** - No token exchange with third-party services
- **Customization** - Adjust permissions as needed for your organization

## Prerequisites

- [GitHub CLI](https://cli.github.com/) installed and authenticated
- `gh-app-create` extension: `gh extension install kennyg/gh-app-create`
- For GHES: Network access to your GitHub Enterprise instance

## Step 1: Create the GitHub App

### For GitHub.com

```bash
gh app-create create --preset opencode
```

### For GitHub Enterprise Server

```bash
gh app-create create --preset opencode --hostname github.mycompany.com
```

### For an Organization

```bash
gh app-create create --preset opencode --org my-org
```

This creates an app named `{username}-opencode-agent` (or `{org}-opencode-agent`) with the required permissions:

| Permission | Access | Purpose |
|------------|--------|---------|
| Contents | Write | Read code, create branches, push commits |
| Pull Requests | Write | Create and update PRs, post comments |
| Issues | Write | Read issues, post comments |
| Metadata | Read | Access repository metadata |

Save the output credentials securely. You'll need:
- **App ID** - Numeric identifier for the app
- **Private Key** - PEM file for authentication

## Step 2: Install the App on Repositories

After creation, install the app on the repositories where you want OpenCode to operate:

1. Go to your app's settings:
   - GitHub.com: `https://github.com/settings/apps/{app-slug}/installations`
   - GHES: `https://github.mycompany.com/settings/apps/{app-slug}/installations`
   - Organization: `https://github.com/organizations/{org}/settings/apps/{app-slug}/installations`

2. Click "Install" and select the repositories

## Step 3: Configure Repository Secrets

Add these secrets to your repository (or organization) under **Settings > Secrets and variables > Actions**:

| Secret | Value |
|--------|-------|
| `APP_ID` | Your GitHub App's numeric ID |
| `APP_PRIVATE_KEY` | The entire PEM file contents (including BEGIN/END lines) |
| `ANTHROPIC_API_KEY` | Your Anthropic API key for Claude |

## Step 4: Create the Workflow

Add `.github/workflows/opencode.yml` to your repository:

```yaml
name: opencode

on:
  issue_comment:
    types: [created]
  pull_request_review_comment:
    types: [created]

jobs:
  opencode:
    if: |
      !endsWith(github.event.comment.user.login, '[bot]') &&
      contains(fromJSON('["OWNER", "MEMBER", "COLLABORATOR"]'), github.event.comment.author_association) &&
      (contains(github.event.comment.body, '/opencode') || contains(github.event.comment.body, '/oc'))
    runs-on: ubuntu-latest
    steps:
      - name: Generate GitHub App Token
        id: app-token
        uses: actions/create-github-app-token@v1
        with:
          app-id: ${{ secrets.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}

      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          token: ${{ steps.app-token.outputs.token }}
          fetch-depth: 1

      - name: Install opencode
        run: curl -fsSL https://opencode.ai/install | bash

      - name: Run opencode
        run: ~/.opencode/bin/opencode github run
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
          MODEL: anthropic/claude-sonnet-4-20250514
          USE_GITHUB_TOKEN: "true"
```

### Key Configuration Details

1. **Security filters** - Prevents bot loops and restricts to authorized users (OWNER, MEMBER, COLLABORATOR)
2. **`actions/create-github-app-token`** - Generates an installation token from your app credentials
3. **Inlined installation** - Installs opencode directly instead of using the `sst/opencode/github` action, avoiding marketplace restrictions
4. **`USE_GITHUB_TOKEN: "true"`** - Tells OpenCode to use the provided `GITHUB_TOKEN` instead of OIDC exchange
5. **No `id-token: write`** - Not needed since we're not using OIDC

### How opencode github run Works

Note that `opencode github run` takes no arguments - it automatically reads the triggering comment from the GitHub Actions event payload (`$GITHUB_EVENT_PATH`). This includes:

- The comment body (your `/opencode` command and prompt)
- Issue or PR context (number, title, body, labels)
- Repository information
- User who triggered the command

Configuration is passed via environment variables (`MODEL`, `USE_GITHUB_TOKEN`, etc.) rather than CLI flags when running in GitHub mode.

For the full event payload schema, see the GitHub docs:
- [issue_comment](https://docs.github.com/en/webhooks/webhook-events-and-payloads#issue_comment)
- [pull_request_review_comment](https://docs.github.com/en/webhooks/webhook-events-and-payloads#pull_request_review_comment)

## Step 5: Test It

1. Open an issue or PR in your repository
2. Comment `/opencode explain this codebase`
3. The workflow should trigger and OpenCode will respond

## GitHub Enterprise Server Notes

### Workflow Adjustments for GHES

If your GHES instance uses a custom API URL, you may need to configure the token action:

```yaml
      - name: Generate GitHub App Token
        id: app-token
        uses: actions/create-github-app-token@v1
        with:
          app-id: ${{ secrets.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}
          github-api-url: https://github.mycompany.com/api/v3
```

### Self-Hosted Runners

For GHES with self-hosted runners, ensure the runner has:
- Network access to your GHES instance
- Network access to Anthropic's API (or your AI provider)
- The `opencode` binary accessible (the action handles installation)

## Troubleshooting

### "Resource not accessible by integration"

The app isn't installed on the repository, or lacks the required permissions. Reinstall the app and ensure all permissions are granted.

### "Bad credentials"

The `APP_PRIVATE_KEY` secret may be malformed. Ensure you copied the entire PEM file including:
```
-----BEGIN RSA PRIVATE KEY-----
...
-----END RSA PRIVATE KEY-----
```

### Workflow doesn't trigger

- Ensure the workflow file is on the default branch
- Check that the comment contains `/opencode` or `/oc`
- Verify the app is installed on the repository

## Using Different AI Providers

OpenCode supports multiple providers. Update the `model` and API key accordingly:

```yaml
# OpenAI
env:
  OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
with:
  model: openai/gpt-4o

# AWS Bedrock
env:
  AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
  AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
  AWS_REGION: us-east-1
with:
  model: bedrock/anthropic.claude-sonnet-4-20250514-v1:0
```

## Using GitHub Copilot via LiteLLM Proxy

If your organization has GitHub Copilot licenses, you can use Copilot's models as the backend for OpenCode via a [LiteLLM](https://docs.litellm.ai/) proxy. This is useful for enterprises that want to leverage existing Copilot licenses instead of separate API keys.

### How It Works

1. Authenticate with GitHub Copilot once locally (OAuth device flow)
2. Cache the tokens and seed them as secrets to your GHE instance
3. In CI, restore tokens and run LiteLLM as a sidecar
4. OpenCode sends requests to LiteLLM, which forwards to Copilot's API

### Prerequisites

- Active GitHub Copilot subscription on github.com
- `gh` CLI authenticated to both github.com and your GHE instance
- LiteLLM: `pip install 'litellm[proxy]'`

### Quick Start: Seed Copilot Tokens to GHE

Use the included script to authenticate and seed tokens:

```bash
# For a specific repository
./scripts/seed-copilot-secrets.sh \
  --repo myorg/myrepo \
  --hostname github.mycompany.com

# For an entire organization
./scripts/seed-copilot-secrets.sh \
  --org myorg \
  --hostname github.mycompany.com
```

This script:
1. Ensures you're authenticated to github.com (for Copilot) and your GHE instance
2. Runs LiteLLM to trigger the OAuth device flow
3. Extracts the cached tokens
4. Uses `gh secret set` to push them to your GHE repo/org

### Manual Setup

If you prefer to set up manually:

**Step 1: Authenticate locally**

```bash
pip install 'litellm[proxy]'

# Create minimal config
cat > /tmp/litellm.yaml <<EOF
model_list:
  - model_name: copilot
    litellm_params:
      model: github_copilot/gpt-4o
EOF

# Start LiteLLM and complete device flow
litellm --config /tmp/litellm.yaml --port 4000
```

On first request, follow the device flow prompts to authenticate.

**Step 2: Extract and seed tokens**

```bash
# Tokens are stored here by default
TOKEN_DIR=~/.config/litellm/github_copilot

# Seed to GHE
gh secret set COPILOT_ACCESS_TOKEN \
  --hostname github.mycompany.com \
  --repo myorg/myrepo \
  < "$TOKEN_DIR/access-token"

gh secret set COPILOT_API_KEY \
  --hostname github.mycompany.com \
  --repo myorg/myrepo \
  < "$TOKEN_DIR/api-key.json"
```

### Workflow Configuration

Add this workflow to `.github/workflows/opencode.yml`:

```yaml
name: opencode

on:
  issue_comment:
    types: [created]
  pull_request_review_comment:
    types: [created]

jobs:
  opencode:
    if: |
      !endsWith(github.event.comment.user.login, '[bot]') &&
      contains(fromJSON('["OWNER", "MEMBER", "COLLABORATOR"]'), github.event.comment.author_association) &&
      (contains(github.event.comment.body, '/opencode') || contains(github.event.comment.body, '/oc'))
    runs-on: ubuntu-latest
    steps:
      - name: Generate GitHub App Token
        id: app-token
        uses: actions/create-github-app-token@v1
        with:
          app-id: ${{ secrets.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}

      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          token: ${{ steps.app-token.outputs.token }}
          fetch-depth: 1

      - name: Setup Copilot tokens
        run: |
          mkdir -p ~/.config/litellm/github_copilot
          echo '${{ secrets.COPILOT_ACCESS_TOKEN }}' > ~/.config/litellm/github_copilot/access-token
          echo '${{ secrets.COPILOT_API_KEY }}' > ~/.config/litellm/github_copilot/api-key.json

      - name: Start LiteLLM proxy
        run: |
          pip install 'litellm[proxy]'
          cat > /tmp/litellm.yaml <<EOF
          model_list:
            - model_name: copilot-claude
              litellm_params:
                model: github_copilot/claude-3.5-sonnet
          general_settings:
            master_key: sk-litellm-key
          EOF
          litellm --config /tmp/litellm.yaml --port 4000 &
          sleep 5  # Wait for proxy to start

      - name: Install opencode
        run: curl -fsSL https://opencode.ai/install | bash

      - name: Run opencode
        run: ~/.opencode/bin/opencode github run
        env:
          GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
          OPENAI_API_KEY: sk-litellm-key
          OPENAI_API_BASE: http://localhost:4000
          MODEL: openai/copilot-claude
          USE_GITHUB_TOKEN: "true"
```

### Required Secrets

| Secret | Source | Description |
|--------|--------|-------------|
| `APP_ID` | `gh-app-create` output | Your GitHub App's numeric ID |
| `APP_PRIVATE_KEY` | `gh-app-create` output | PEM file for GitHub App auth |
| `COPILOT_ACCESS_TOKEN` | `seed-copilot-secrets.sh` | GitHub OAuth token for Copilot |
| `COPILOT_API_KEY` | `seed-copilot-secrets.sh` | Copilot API key JSON |

### Available Copilot Models

GitHub Copilot exposes these models via its API:

| Model | LiteLLM Config |
|-------|----------------|
| GPT-4o | `github_copilot/gpt-4o` |
| GPT-4o Mini | `github_copilot/gpt-4o-mini` |
| Claude 3.5 Sonnet | `github_copilot/claude-3.5-sonnet` |
| o1 | `github_copilot/o1` |
| o1-mini | `github_copilot/o1-mini` |
| o3-mini | `github_copilot/o3-mini` |

### Token Storage

LiteLLM caches Copilot credentials locally. Configure the location with environment variables:

```bash
GITHUB_COPILOT_TOKEN_DIR=/path/to/tokens
GITHUB_COPILOT_ACCESS_TOKEN_FILE=access_token.json
GITHUB_COPILOT_API_KEY_FILE=api_key.json
```

For containerized deployments, mount this directory as a volume to persist authentication across restarts.

### Troubleshooting LiteLLM + Copilot

**"Device flow authentication required"**

The proxy needs initial authentication. Run LiteLLM interactively first to complete the device flow, then deploy with cached credentials.

**"Copilot access denied"**

Ensure your GitHub account has an active Copilot subscription and the OAuth app is authorized.

**Rate limits**

GitHub Copilot has rate limits. Consider configuring LiteLLM with fallback models:

```yaml
model_list:
  - model_name: claude
    litellm_params:
      model: github_copilot/claude-3.5-sonnet
  - model_name: claude
    litellm_params:
      model: anthropic/claude-3-5-sonnet-20241022
      api_key: os.environ/ANTHROPIC_API_KEY
```

This falls back to direct Anthropic API if Copilot is unavailable.

## Exporting Credentials for Automation

When creating the app, export credentials in a format ready for GitHub Actions:

```bash
gh app-create create --preset opencode --format github-actions --output credentials.txt
```

This outputs the values in a format you can copy directly to GitHub Secrets.
