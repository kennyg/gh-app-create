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
    if: contains(github.event.comment.body, '/opencode') || contains(github.event.comment.body, '/oc')
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

      - name: Run opencode
        uses: sst/opencode/github@dev
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
        with:
          model: anthropic/claude-sonnet-4-20250514
          use_github_token: "true"
```

### Key Differences from Public App Setup

1. **`actions/create-github-app-token`** - Generates an installation token from your app credentials
2. **`use_github_token: "true"`** - Tells OpenCode to use the provided `GITHUB_TOKEN` instead of OIDC exchange
3. **No `id-token: write`** - Not needed since we're not using OIDC

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

1. LiteLLM proxy authenticates with GitHub Copilot using OAuth device flow
2. OpenCode sends requests to your LiteLLM proxy
3. LiteLLM forwards them to GitHub Copilot's API (`api.githubcopilot.com`)

### Prerequisites

- Active GitHub Copilot subscription (Individual, Business, or Enterprise)
- LiteLLM proxy running (self-hosted or as a service)

### Step 1: Set Up LiteLLM Proxy

Create `litellm_config.yaml`:

```yaml
model_list:
  - model_name: copilot-claude
    litellm_params:
      model: github_copilot/claude-3.5-sonnet
  - model_name: copilot-gpt4
    litellm_params:
      model: github_copilot/gpt-4o

general_settings:
  master_key: sk-your-master-key-here
```

Run the proxy:

```bash
pip install 'litellm[proxy]'
litellm --config litellm_config.yaml --port 4000
```

On first request, LiteLLM will prompt you to authenticate via GitHub's device flow:
1. Visit the URL shown (e.g., `https://github.com/login/device`)
2. Enter the code displayed
3. Authorize the application
4. Credentials are cached locally for future use

### Step 2: Deploy LiteLLM for GitHub Actions

For GitHub Actions, you'll need LiteLLM accessible from your runners. Options:

**Option A: Self-hosted runner with local proxy**
```yaml
jobs:
  opencode:
    runs-on: self-hosted
    services:
      litellm:
        image: ghcr.io/berriai/litellm:main-latest
        ports:
          - 4000:4000
        volumes:
          - /path/to/litellm_config.yaml:/app/config.yaml
          - /path/to/copilot-tokens:/root/.litellm  # Cached auth tokens
        env:
          LITELLM_MASTER_KEY: ${{ secrets.LITELLM_MASTER_KEY }}
```

**Option B: Centralized LiteLLM service**

Deploy LiteLLM as a service accessible to your runners (e.g., internal Kubernetes service, VM).

### Step 3: Update Workflow

```yaml
name: opencode

on:
  issue_comment:
    types: [created]
  pull_request_review_comment:
    types: [created]

jobs:
  opencode:
    if: contains(github.event.comment.body, '/opencode') || contains(github.event.comment.body, '/oc')
    runs-on: ubuntu-latest  # or self-hosted
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

      - name: Run opencode
        uses: sst/opencode/github@dev
        env:
          GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
          OPENAI_API_KEY: ${{ secrets.LITELLM_MASTER_KEY }}
          OPENAI_API_BASE: ${{ secrets.LITELLM_URL }}  # e.g., https://litellm.internal.company.com
        with:
          model: openai/copilot-claude  # Model name from your litellm_config.yaml
          use_github_token: "true"
```

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
