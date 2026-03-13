package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "shdw",
	Short: "Shadow — a local encrypted secrets manager",
	Long: `Shadow (shdw) is a local-first encrypted secrets manager for developers.
Secrets are stored in an AES-256 encrypted vault on your machine.
No cloud, no account, no telemetry.

Secrets are organised into namespaces. A bare key with no namespace
is stored in the 'global' namespace automatically.

Examples:
  shdw set discord/api_key abc123
  shdw get discord/api_key
  shdw run discord discord/prod -- node app.js
  shdw export discord discord/prod -o .env
  shdw list`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(lockCmd)
	rootCmd.AddCommand(renameCmd)
	rootCmd.AddCommand(copyCmd)
	rootCmd.AddCommand(unlockCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(versionCmd)
}
