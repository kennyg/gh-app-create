package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/browser"
	"github.com/kennyg/gh-app-create/internal/manifest"
	"github.com/kennyg/gh-app-create/internal/server"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	flagOrg      string
	flagPreset   string
	flagManifest string
	flagName     string
	flagOutput   string
	flagFormat   string
	flagPort     int
	flagHostname string
	flagDebug    bool
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new GitHub App from a manifest or preset",
	Long: `Create a new GitHub App using the manifest flow.

This will open your browser to confirm the app creation on GitHub,
then automatically capture and output the credentials.`,
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringVarP(&flagOrg, "org", "o", "", "Organization to create the app in (omit for personal account)")
	createCmd.Flags().StringVarP(&flagPreset, "preset", "p", "", "Use a built-in preset (see 'presets' command)")
	createCmd.Flags().StringVarP(&flagManifest, "manifest", "m", "", "Path to manifest YAML file")
	createCmd.Flags().StringVarP(&flagName, "name", "n", "", "Override the app name")
	createCmd.Flags().StringVarP(&flagOutput, "output", "O", "", "Output file for credentials (default: stdout)")
	createCmd.Flags().StringVarP(&flagFormat, "format", "f", "yaml", "Output format: yaml, json, env, github-actions")
	createCmd.Flags().IntVar(&flagPort, "port", 9876, "Local port for OAuth callback")
	createCmd.Flags().StringVar(&flagHostname, "hostname", "", "GitHub Enterprise Server hostname")
	createCmd.Flags().BoolVar(&flagDebug, "debug", false, "Show manifest JSON in the browser form")
}

func runCreate(cmd *cobra.Command, args []string) error {
	var m manifest.Manifest

	if flagPreset != "" {
		var ok bool
		m, ok = manifest.Presets[flagPreset]
		if !ok {
			return fmt.Errorf("unknown preset: %s\nRun 'gh app-create presets' to see available options", flagPreset)
		}
		fmt.Fprintf(os.Stderr, "Using preset: %s\n", flagPreset)
	} else if flagManifest != "" {
		data, err := os.ReadFile(flagManifest)
		if err != nil {
			return fmt.Errorf("reading manifest file: %w", err)
		}
		if err := yaml.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("parsing manifest YAML: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Using manifest: %s\n", flagManifest)
	} else {
		return fmt.Errorf("must specify --preset or --manifest\nRun 'gh app-create presets' to see available presets")
	}

	host := "github.com"
	if flagHostname != "" {
		host = flagHostname
	}

	// Determine the app name prefix (org or username) for uniqueness
	if flagName != "" {
		m.Name = flagName
	} else if flagPreset != "" {
		// Auto-prefix preset names with org or username to avoid slug conflicts
		prefix := flagOrg
		if prefix == "" {
			prefix = getGitHubUsername(host)
		}
		if prefix != "" {
			m.Name = fmt.Sprintf("%s-%s", prefix, m.Name)
		}
	}

	m.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", flagPort)

	// If manifest has events but no webhook config, add a disabled webhook placeholder
	// GitHub requires hook_attributes when events are specified
	if len(m.DefaultEvents) > 0 && m.HookAttributes == nil {
		m.HookAttributes = &manifest.HookAttributes{
			URL:    "https://example.com/webhook",
			Active: false,
		}
	}

	var registerURL string
	if flagOrg != "" {
		registerURL = fmt.Sprintf("https://%s/organizations/%s/settings/apps/new", host, flagOrg)
	} else {
		registerURL = fmt.Sprintf("https://%s/settings/apps/new", host)
	}

	fmt.Fprintf(os.Stderr, "\nCreating GitHub App: %s\n", m.Name)
	fmt.Fprintf(os.Stderr, "   Target: %s\n", registerURL)
	fmt.Fprintf(os.Stderr, "   Permissions: %v\n", manifest.FormatPermissions(m.DefaultPermissions))
	fmt.Fprintf(os.Stderr, "   Events: %v\n\n", m.DefaultEvents)

	manifestJSON, _ := json.Marshal(m)
	prettyJSON, _ := json.MarshalIndent(m, "", "  ")
	formHTML := buildFormHTML(m, registerURL, string(manifestJSON), string(prettyJSON), flagDebug)

	callbackServer := server.New(flagPort)
	callbackServer.FormHTML = formHTML
	if err := callbackServer.Start(); err != nil {
		return fmt.Errorf("starting callback server: %w", err)
	}
	defer callbackServer.Close()

	localURL := fmt.Sprintf("http://localhost:%d", flagPort)
	fmt.Fprintf(os.Stderr, "Opening browser...\n")
	fmt.Fprintf(os.Stderr, "   (If browser doesn't open, visit: %s)\n\n", localURL)

	b := browser.New("", os.Stdout, os.Stderr)
	_ = b.Browse(localURL)

	fmt.Fprintf(os.Stderr, "Waiting for GitHub callback on port %d...\n", flagPort)

	code, err := callbackServer.WaitForCode(5 * time.Minute)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Received authorization\n")

	fmt.Fprintf(os.Stderr, "Exchanging code for credentials...\n")

	opts := api.ClientOptions{}
	if flagHostname != "" {
		opts.Host = flagHostname
	}

	client, err := api.NewRESTClient(opts)
	if err != nil {
		return fmt.Errorf("creating API client: %w", err)
	}

	var response struct {
		ID            int64  `json:"id"`
		Slug          string `json:"slug"`
		ClientID      string `json:"client_id"`
		ClientSecret  string `json:"client_secret"`
		PEM           string `json:"pem"`
		WebhookSecret string `json:"webhook_secret"`
	}

	err = client.Post(fmt.Sprintf("app-manifests/%s/conversions", code), nil, &response)
	if err != nil {
		return fmt.Errorf("exchanging code for credentials: %w", err)
	}

	creds := manifest.Credentials{
		AppID:         response.ID,
		AppSlug:       response.Slug,
		ClientID:      response.ClientID,
		ClientSecret:  response.ClientSecret,
		PEM:           response.PEM,
		WebhookSecret: response.WebhookSecret,
	}

	output, err := manifest.FormatCredentials(creds, flagFormat)
	if err != nil {
		return fmt.Errorf("formatting credentials: %w", err)
	}

	if flagOutput != "" {
		if err := os.WriteFile(flagOutput, output, 0600); err != nil {
			return fmt.Errorf("writing credentials: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\nGitHub App created successfully!\n")
		fmt.Fprintf(os.Stderr, "   App ID:   %d\n", creds.AppID)
		fmt.Fprintf(os.Stderr, "   Slug:     %s\n", creds.AppSlug)
		fmt.Fprintf(os.Stderr, "   Saved to: %s\n", flagOutput)
	} else {
		fmt.Fprintf(os.Stderr, "\nGitHub App created successfully!\n")
		fmt.Fprintf(os.Stderr, "   App ID: %d\n", creds.AppID)
		fmt.Fprintf(os.Stderr, "   Slug:   %s\n\n", creds.AppSlug)
		fmt.Fprintf(os.Stderr, "--- Credentials (keep these safe!) ---\n")
		fmt.Println(string(output))
	}

	var installURL string
	if flagOrg != "" {
		installURL = fmt.Sprintf("https://%s/organizations/%s/settings/apps/%s/installations", host, flagOrg, creds.AppSlug)
	} else {
		installURL = fmt.Sprintf("https://%s/settings/apps/%s/installations", host, creds.AppSlug)
	}

	fmt.Fprintf(os.Stderr, "\nNext steps:\n")
	fmt.Fprintf(os.Stderr, "   1. Install the app on repositories:\n")
	fmt.Fprintf(os.Stderr, "      %s\n", installURL)
	fmt.Fprintf(os.Stderr, "   2. Use 'gh token' extension to generate tokens:\n")
	fmt.Fprintf(os.Stderr, "      gh extension install Link-/gh-token\n")
	fmt.Fprintf(os.Stderr, "      gh token generate --app-id %d --key ./private-key.pem\n", creds.AppID)

	return nil
}

func buildFormHTML(m manifest.Manifest, registerURL, manifestJSON, prettyJSON string, debug bool) string {
	debugPanel := ""
	if debug {
		debugPanel = fmt.Sprintf(`<details class="debug">
<summary>Manifest JSON</summary>
<pre>%s</pre>
</details>`, prettyJSON)
	}

	eventsHTML := ""
	if len(m.DefaultEvents) > 0 {
		eventsHTML = `<div class="section"><h4>Events</h4><div class="badges">`
		for _, event := range m.DefaultEvents {
			eventsHTML += fmt.Sprintf(`<span class="badge badge-event">%s</span>`, event)
		}
		eventsHTML += `</div></div>`
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<title>Create GitHub App: %s</title>
<style>
* { box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: #f6f8fa; padding: 20px; }
.card { background: white; padding: 32px; border-radius: 12px; box-shadow: 0 4px 12px rgba(0,0,0,0.1); text-align: center; max-width: 480px; width: 100%%; }
h2 { margin: 0 0 8px 0; color: #24292f; font-size: 24px; }
.app-name { font-size: 18px; font-weight: 600; color: #24292f; margin: 0 0 4px 0; }
.app-desc { color: #57606a; margin: 0 0 20px 0; font-size: 14px; }
.section { text-align: left; background: #f6f8fa; padding: 16px; border-radius: 8px; margin-bottom: 16px; }
.section h4 { margin: 0 0 12px 0; color: #24292f; font-size: 13px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }
.badges { display: flex; flex-wrap: wrap; gap: 8px; }
.badge { display: inline-flex; align-items: center; padding: 6px 10px; border-radius: 6px; font-size: 12px; font-weight: 500; }
.badge-read { background: #ddf4ff; color: #0969da; }
.badge-write { background: #fff8c5; color: #9a6700; }
.badge-event { background: #f3e8ff; color: #8250df; }
button { background: #2da44e; color: white; border: none; padding: 14px 28px; font-size: 16px; font-weight: 500; border-radius: 8px; cursor: pointer; width: 100%%; transition: background 0.15s; }
button:hover { background: #22863a; }
.debug { text-align: left; margin-top: 16px; border: 1px dashed #d0d7de; border-radius: 8px; padding: 12px; background: #f6f8fa; }
.debug summary { cursor: pointer; font-size: 12px; color: #57606a; font-weight: 500; }
.debug pre { margin: 12px 0 0 0; padding: 12px; background: #24292f; color: #7ee787; border-radius: 6px; font-size: 11px; overflow-x: auto; white-space: pre-wrap; word-break: break-all; }
</style>
</head>
<body>
<div class="card">
<h2>Create GitHub App</h2>
<p class="app-name">%s</p>
<p class="app-desc">%s</p>
<div class="section">
<h4>Permissions</h4>
<div class="badges">%s</div>
</div>
%s
<form action="%s" method="post">
<input type="hidden" name="manifest" value='%s'>
<button type="submit">Create GitHub App</button>
</form>
%s
</div>
</body>
</html>`,
		m.Name,
		m.Name,
		m.Description,
		manifest.FormatPermissionsHTML(m.DefaultPermissions),
		eventsHTML,
		registerURL,
		manifestJSON,
		debugPanel,
	)
}

// getGitHubUsername fetches the authenticated user's login
func getGitHubUsername(host string) string {
	opts := api.ClientOptions{}
	if host != "github.com" {
		opts.Host = host
	}
	client, err := api.NewRESTClient(opts)
	if err != nil {
		return ""
	}
	var user struct {
		Login string `json:"login"`
	}
	if err := client.Get("user", &user); err != nil {
		return ""
	}
	return user.Login
}
