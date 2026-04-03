package cmd

import (
	"fmt"
	"os"

	"github.com/leihog/shdw/internal/store"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:     "rename <old-path> <new-path>",
	Aliases: []string{"mv"},
	Short:   "Rename or move a key or namespace",
	Long: `Rename or move a key or namespace to a new path.

Works for both keys and namespaces. When moving a namespace, all keys
within it are moved along with it.
Intermediate namespaces at the destination are created automatically.`,
	Example: `  shdw rename token global_token
  shdw rename discord/api_key discord/prod/api_key
  shdw rename discord services/discord`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: anyCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		oldPath, newPath := args[0], args[1]

		password, err := getMasterPassword(false)
		if err != nil {
			return err
		}

		vault, err := openVault(password)
		if err != nil {
			return err
		}

		if err := vault.Rename(oldPath, newPath); err != nil {
			return err
		}

		if err := store.Save(vault, password); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Renamed '%s' -> '%s'.\n", oldPath, newPath)
		return nil
	},
}
