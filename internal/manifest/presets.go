package manifest

// Presets contains built-in GitHub App configurations
var Presets = map[string]Manifest{
	"opencode": {
		Name:        "opencode-agent",
		URL:         "https://opencode.ai",
		Description: "AI coding agent for GitHub issues and PRs",
		Public:      false,
		DefaultPermissions: map[string]string{
			"contents":      "write",
			"pull_requests": "write",
			"issues":        "write",
			"metadata":      "read",
		},
		DefaultEvents: []string{
			"issue_comment",
			"pull_request_review_comment",
		},
	},
	"ci-bot": {
		Name:        "ci-bot",
		URL:         "https://example.com",
		Description: "CI/CD automation bot",
		Public:      false,
		DefaultPermissions: map[string]string{
			"contents":      "write",
			"pull_requests": "write",
			"checks":        "write",
			"statuses":      "write",
			"metadata":      "read",
		},
		DefaultEvents: []string{
			"push",
			"pull_request",
			"check_run",
			"check_suite",
		},
	},
	"release-bot": {
		Name:        "release-bot",
		URL:         "https://example.com",
		Description: "Automated release management",
		Public:      false,
		DefaultPermissions: map[string]string{
			"contents":      "write",
			"pull_requests": "write",
			"issues":        "write",
			"metadata":      "read",
		},
		DefaultEvents: []string{
			"release",
			"push",
		},
	},
	"readonly": {
		Name:        "readonly-app",
		URL:         "https://example.com",
		Description: "Read-only access for analysis tools",
		Public:      false,
		DefaultPermissions: map[string]string{
			"contents": "read",
			"metadata": "read",
		},
		DefaultEvents: []string{},
	},
	"actions-token": {
		Name:        "actions-token-app",
		URL:         "https://example.com",
		Description: "GitHub App for generating installation tokens in Actions",
		Public:      false,
		DefaultPermissions: map[string]string{
			"contents":      "write",
			"pull_requests": "write",
			"issues":        "write",
			"actions":       "write",
			"metadata":      "read",
		},
		DefaultEvents: []string{},
	},
	"renovate": {
		Name:        "renovate-bot",
		URL:         "https://github.com/renovatebot/renovate",
		Description: "Self-hosted Renovate for automated dependency updates",
		Public:      false,
		DefaultPermissions: map[string]string{
			"checks":            "write",
			"statuses":          "write",
			"contents":          "write",
			"issues":            "write",
			"pull_requests":     "write",
			"workflows":         "write",
			"administration":    "read",
			"dependabot_alerts": "read",
			"members":           "read",
			"metadata":          "read",
		},
		DefaultEvents: []string{
			"check_run",
			"issue_comment",
			"pull_request",
			"push",
		},
	},
}

// BlankTemplate returns an empty manifest template for customization
func BlankTemplate() Manifest {
	return Manifest{
		Name:        "my-github-app",
		URL:         "https://example.com",
		Description: "My custom GitHub App",
		Public:      false,
		DefaultPermissions: map[string]string{
			"contents": "read",
			"metadata": "read",
		},
		DefaultEvents: []string{},
	}
}
