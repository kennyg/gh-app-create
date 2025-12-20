package commands

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh-app-create",
	Short: "Create and manage GitHub Apps via manifests",
	Long: `A gh CLI extension for creating GitHub Apps programmatically.

Uses the GitHub App Manifest flow to create apps with predefined
permissions, eliminating manual configuration through the web UI.

Examples:
  # Create from built-in preset
  gh app-create create --preset opencode --org my-org

  # Create from custom manifest file
  gh app-create create --manifest ./my-app.yaml --org my-org

  # For GitHub Enterprise Server
  gh app-create create --preset ci-bot --org my-org --hostname github.mycompany.com

  # List available presets
  gh app-create presets`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(presetsCmd)
	rootCmd.AddCommand(exportCmd)
}
