package manifest

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest represents a GitHub App manifest
type Manifest struct {
	Name               string            `json:"name" yaml:"name"`
	URL                string            `json:"url" yaml:"url"`
	HookAttributes     *HookAttributes   `json:"hook_attributes,omitempty" yaml:"hook_attributes,omitempty"`
	RedirectURL        string            `json:"redirect_url,omitempty" yaml:"redirect_url,omitempty"`
	CallbackURLs       []string          `json:"callback_urls,omitempty" yaml:"callback_urls,omitempty"`
	SetupURL           string            `json:"setup_url,omitempty" yaml:"setup_url,omitempty"`
	Description        string            `json:"description,omitempty" yaml:"description,omitempty"`
	Public             bool              `json:"public" yaml:"public"`
	DefaultPermissions map[string]string `json:"default_permissions" yaml:"default_permissions"`
	DefaultEvents      []string          `json:"default_events,omitempty" yaml:"default_events,omitempty"`
}

// HookAttributes configures webhooks for the app
type HookAttributes struct {
	URL    string `json:"url,omitempty" yaml:"url,omitempty"`
	Active bool   `json:"active" yaml:"active"`
}

// Credentials holds the created app's secrets
type Credentials struct {
	AppID         int64  `json:"app_id" yaml:"app_id"`
	AppSlug       string `json:"app_slug" yaml:"app_slug"`
	ClientID      string `json:"client_id" yaml:"client_id"`
	ClientSecret  string `json:"client_secret" yaml:"client_secret"`
	PEM           string `json:"pem" yaml:"pem"`
	WebhookSecret string `json:"webhook_secret" yaml:"webhook_secret"`
}

// FormatPermissions returns a human-readable string of permissions
func FormatPermissions(perms map[string]string) string {
	parts := make([]string, 0, len(perms))
	for k, v := range perms {
		parts = append(parts, fmt.Sprintf("%s:%s", k, v))
	}
	return strings.Join(parts, ", ")
}

// FormatPermissionsHTML returns permissions formatted as HTML badges
func FormatPermissionsHTML(perms map[string]string) string {
	var sb strings.Builder
	for k, v := range perms {
		badgeClass := "badge-read"
		if v == "write" {
			badgeClass = "badge-write"
		}
		sb.WriteString(fmt.Sprintf(`<span class="badge %s">%s: %s</span>`, badgeClass, k, v))
	}
	return sb.String()
}

// FormatCredentials formats credentials in the specified format
func FormatCredentials(creds Credentials, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(creds, "", "  ")
	case "env":
		return []byte(fmt.Sprintf(`GITHUB_APP_ID=%d
GITHUB_APP_SLUG=%s
GITHUB_APP_CLIENT_ID=%s
GITHUB_APP_CLIENT_SECRET=%s
GITHUB_APP_WEBHOOK_SECRET=%s
GITHUB_APP_PRIVATE_KEY=%s
`, creds.AppID, creds.AppSlug, creds.ClientID, creds.ClientSecret, creds.WebhookSecret, escapeForEnv(creds.PEM))), nil
	case "github-actions":
		return []byte(fmt.Sprintf(`# Add these to your repository or organization secrets:
# Settings → Secrets and variables → Actions → New repository secret

# Secret name: APP_ID
%d

# Secret name: APP_PRIVATE_KEY (copy the entire block including BEGIN/END lines)
%s
`, creds.AppID, creds.PEM)), nil
	default: // yaml
		return yaml.Marshal(creds)
	}
}

func escapeForEnv(s string) string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "'\\''"))
}
