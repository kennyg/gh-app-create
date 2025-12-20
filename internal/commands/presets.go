package commands

import (
	"fmt"
	"strings"

	"github.com/kennyg/gh-app-create/internal/manifest"
	"github.com/spf13/cobra"
)

var presetsCmd = &cobra.Command{
	Use:   "presets",
	Short: "List available presets",
	Run:   runPresets,
}

func runPresets(cmd *cobra.Command, args []string) {
	fmt.Println("Available presets:")
	fmt.Println()
	for name, m := range manifest.Presets {
		fmt.Printf("  \033[1m%s\033[0m\n", name)
		fmt.Printf("    %s\n", m.Description)
		fmt.Printf("    Permissions: %s\n", manifest.FormatPermissions(m.DefaultPermissions))
		if len(m.DefaultEvents) > 0 {
			fmt.Printf("    Events: %s\n", strings.Join(m.DefaultEvents, ", "))
		}
		fmt.Println()
	}
	fmt.Println("Usage: gh app-create create --preset <name> [--org <org>]")
}
