package cmd

import (
	"fmt"
	"os"

	"github.com/leihog/shdw/internal/keychain"
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Clear the cached master password from the OS keychain",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := keychain.Delete(); err != nil {
			fmt.Fprintln(os.Stderr, "Nothing to clear (vault was already locked).")
			return nil
		}
		fmt.Fprintln(os.Stderr, "Vault locked. Password cleared from keychain.")
		return nil
	},
}
