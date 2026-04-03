package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock the vault by caching the master password in the OS keychain",
	Long: `Prompt for the master password and cache it in the OS keychain.

This is optional — any command that needs the vault will prompt automatically.
Use this when you want to pre-unlock without running another command.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		password, err := getMasterPassword(false)
		if err != nil {
			return err
		}
		if _, err := openVault(password); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Vault unlocked.")
		return nil
	},
}
