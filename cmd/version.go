package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags.
// Falls back to "dev" when built without version injection.
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of shdw",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Shadow (shdw) — local encrypted secrets manager")
		fmt.Println("────────────────────────────────────────────────")
		fmt.Printf("  version  %s\n", version)
		fmt.Printf("  author   Leif Högberg\n")
		fmt.Printf("  repo     https://github.com/leihog/shdw\n")
	},
}
