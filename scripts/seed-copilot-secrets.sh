#!/usr/bin/env bash
set -euo pipefail

# seed-copilot-secrets.sh
# Authenticates with GitHub Copilot via LiteLLM and seeds the tokens as secrets
# to a GitHub Enterprise Server repository or organization.
#
# Prerequisites:
#   - gh CLI installed and authenticated to both github.com and your GHE instance
#   - Active GitHub Copilot subscription on github.com
#   - pip install 'litellm[proxy]'
#
# Usage:
#   ./seed-copilot-secrets.sh --repo owner/repo --hostname github.mycompany.com
#   ./seed-copilot-secrets.sh --org my-org --hostname github.mycompany.com

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOKEN_DIR="${GITHUB_COPILOT_TOKEN_DIR:-$HOME/.config/litellm/github_copilot}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Authenticate with GitHub Copilot and seed tokens as secrets to GHE.

Options:
    --repo OWNER/REPO     Target repository for secrets
    --org ORG             Target organization for secrets
    --hostname HOST       GHE hostname (e.g., github.mycompany.com)
    --token-dir DIR       Custom token directory (default: ~/.config/litellm/github_copilot)
    --skip-auth           Skip Copilot authentication (use existing tokens)
    --help                Show this help message

Examples:
    # Seed secrets to a specific repo
    $(basename "$0") --repo myorg/myrepo --hostname github.mycompany.com

    # Seed secrets to an organization (available to all repos)
    $(basename "$0") --org myorg --hostname github.mycompany.com

    # Use existing tokens without re-authenticating
    $(basename "$0") --repo myorg/myrepo --hostname github.mycompany.com --skip-auth
EOF
    exit 0
}

log() { echo -e "${GREEN}==>${NC} $*"; }
warn() { echo -e "${YELLOW}Warning:${NC} $*" >&2; }
error() { echo -e "${RED}Error:${NC} $*" >&2; exit 1; }

# Parse arguments
REPO=""
ORG=""
GHE_HOSTNAME=""
SKIP_AUTH=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --repo) REPO="$2"; shift 2 ;;
        --org) ORG="$2"; shift 2 ;;
        --hostname) GHE_HOSTNAME="$2"; shift 2 ;;
        --token-dir) TOKEN_DIR="$2"; shift 2 ;;
        --skip-auth) SKIP_AUTH=true; shift ;;
        --help) usage ;;
        *) error "Unknown option: $1" ;;
    esac
done

# Validate arguments
[[ -z "$GHE_HOSTNAME" ]] && error "Missing required --hostname"
[[ -z "$REPO" && -z "$ORG" ]] && error "Must specify either --repo or --org"
[[ -n "$REPO" && -n "$ORG" ]] && error "Cannot specify both --repo and --org"

# Check prerequisites
command -v gh >/dev/null 2>&1 || error "gh CLI not found. Install from https://cli.github.com"
command -v litellm >/dev/null 2>&1 || error "litellm not found. Install with: pip install 'litellm[proxy]'"

# Step 1: Ensure gh is authenticated to github.com
log "Checking github.com authentication..."
if ! gh auth status --hostname github.com >/dev/null 2>&1; then
    warn "Not authenticated to github.com"
    log "Authenticating to github.com (required for Copilot)..."
    gh auth login --hostname github.com
fi

GITHUB_USER=$(gh api user --hostname github.com --jq '.login')
log "Authenticated to github.com as: $GITHUB_USER"

# Step 2: Ensure gh is authenticated to GHE
log "Checking $GHE_HOSTNAME authentication..."
if ! gh auth status --hostname "$GHE_HOSTNAME" >/dev/null 2>&1; then
    warn "Not authenticated to $GHE_HOSTNAME"
    log "Authenticating to $GHE_HOSTNAME..."
    gh auth login --hostname "$GHE_HOSTNAME"
fi

GHE_USER=$(gh api user --hostname "$GHE_HOSTNAME" --jq '.login')
log "Authenticated to $GHE_HOSTNAME as: $GHE_USER"

# Step 3: Authenticate with Copilot via LiteLLM (unless skipped)
if [[ "$SKIP_AUTH" == "false" ]]; then
    log "Authenticating with GitHub Copilot via LiteLLM..."
    log "This will open a browser for GitHub device flow authentication."
    echo ""

    mkdir -p "$TOKEN_DIR"
    export GITHUB_COPILOT_TOKEN_DIR="$TOKEN_DIR"

    # Create a minimal config to trigger auth
    TEMP_CONFIG=$(mktemp)
    cat > "$TEMP_CONFIG" <<EOF
model_list:
  - model_name: copilot-test
    litellm_params:
      model: github_copilot/gpt-4o
EOF

    # Start litellm briefly to trigger auth, then make a test request
    log "Starting LiteLLM to trigger authentication..."
    litellm --config "$TEMP_CONFIG" --port 14141 &
    LITELLM_PID=$!
    sleep 3

    # Make a test request to trigger the auth flow
    log "Triggering Copilot authentication (follow the prompts)..."
    curl -s -X POST http://localhost:14141/v1/chat/completions \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer sk-test" \
        -d '{"model": "copilot-test", "messages": [{"role": "user", "content": "test"}]}' \
        > /dev/null 2>&1 || true

    # Give time for auth to complete
    sleep 5

    # Cleanup
    kill $LITELLM_PID 2>/dev/null || true
    rm -f "$TEMP_CONFIG"

    echo ""
fi

# Step 4: Verify tokens exist
ACCESS_TOKEN_FILE="$TOKEN_DIR/access-token"
API_KEY_FILE="$TOKEN_DIR/api-key.json"

[[ ! -f "$ACCESS_TOKEN_FILE" ]] && error "Access token not found at $ACCESS_TOKEN_FILE. Run without --skip-auth to authenticate."
[[ ! -f "$API_KEY_FILE" ]] && error "API key not found at $API_KEY_FILE. Run without --skip-auth to authenticate."

log "Found Copilot tokens in $TOKEN_DIR"

# Step 5: Read tokens
ACCESS_TOKEN=$(cat "$ACCESS_TOKEN_FILE")
API_KEY=$(cat "$API_KEY_FILE")

# Step 6: Set secrets on GHE
log "Setting secrets on $GHE_HOSTNAME..."

if [[ -n "$REPO" ]]; then
    log "Target: repository $REPO"

    echo "$ACCESS_TOKEN" | gh secret set COPILOT_ACCESS_TOKEN \
        --hostname "$GHE_HOSTNAME" \
        --repo "$REPO"
    log "Set COPILOT_ACCESS_TOKEN"

    echo "$API_KEY" | gh secret set COPILOT_API_KEY \
        --hostname "$GHE_HOSTNAME" \
        --repo "$REPO"
    log "Set COPILOT_API_KEY"
else
    log "Target: organization $ORG"

    echo "$ACCESS_TOKEN" | gh secret set COPILOT_ACCESS_TOKEN \
        --hostname "$GHE_HOSTNAME" \
        --org "$ORG" \
        --visibility all
    log "Set COPILOT_ACCESS_TOKEN (org-wide)"

    echo "$API_KEY" | gh secret set COPILOT_API_KEY \
        --hostname "$GHE_HOSTNAME" \
        --org "$ORG" \
        --visibility all
    log "Set COPILOT_API_KEY (org-wide)"
fi

echo ""
log "Done! Copilot tokens have been seeded to $GHE_HOSTNAME"
echo ""
echo "Next steps:"
echo "  1. Add LiteLLM to your workflow (see docs/opencode-agent-setup.md)"
echo "  2. Configure your workflow to restore tokens before starting LiteLLM:"
echo ""
echo "     - name: Setup Copilot tokens"
echo "       run: |"
echo "         mkdir -p ~/.config/litellm/github_copilot"
echo '         echo "${{ secrets.COPILOT_ACCESS_TOKEN }}" > ~/.config/litellm/github_copilot/access-token'
echo '         echo "${{ secrets.COPILOT_API_KEY }}" > ~/.config/litellm/github_copilot/api-key.json'
echo ""
