package commands

import (
	"fmt"
	"os"

	"github.com/kennyg/gh-app-create/internal/manifest"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	exportPreset string
	exportOutput string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a preset or blank manifest to a file",
	Long: `Export a manifest template for customization.

Examples:
  gh app-create export --preset opencode --output my-app.yaml
  gh app-create export --output blank.yaml`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportPreset, "preset", "p", "", "Preset to export (blank template if omitted)")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "O", "", "Output file (default: stdout)")
}

func runExport(cmd *cobra.Command, args []string) error {
	var m manifest.Manifest

	if exportPreset != "" {
		var ok bool
		m, ok = manifest.Presets[exportPreset]
		if !ok {
			return fmt.Errorf("unknown preset: %s", exportPreset)
		}
	} else {
		m = manifest.BlankTemplate()
	}

	output, _ := yaml.Marshal(m)

	if exportOutput != "" {
		if err := os.WriteFile(exportOutput, output, 0644); err != nil {
			return fmt.Errorf("writing manifest: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Manifest exported to: %s\n", exportOutput)
	} else {
		fmt.Println(string(output))
	}
	return nil
}
