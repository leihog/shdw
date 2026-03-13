package cmd

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	"github.com/leihog/shdw/internal/store"
	"github.com/spf13/cobra"
)

var copyCmd = &cobra.Command{
	Use:     "copy <path>",
	Aliases: []string{"cp"},
	Short:   "Copy a secret value to the clipboard",
	Long: `Copy a secret value to the clipboard without printing it to the terminal.

The value is never written to stdout or stderr.`,
	Example: `  shdw copy discord/api_key
  shdw cp token`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: keyCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		password, err := getMasterPassword(false)
		if err != nil {
			return err
		}

		vault, err := store.Load(password)
		if err != nil {
			return err
		}

		secret, err := vault.Get(path)
		if err != nil {
			return err
		}

		if err := clipboard.WriteAll(secret.Value); err != nil {
			return fmt.Errorf("could not write to clipboard: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Copied '%s' to clipboard.\n", path)
		return nil
	},
}
