package cmd

import (
	"fmt"
	"os"

	"github.com/leihog/shdw/internal/store"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <path>",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a key or namespace",
	Long: `Delete a key or an entire namespace (and everything inside it).

Use with care — deleting a namespace removes all keys within it.`,
	Example: `  shdw delete token
  shdw delete discord/api_key
  shdw delete discord`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: anyCompleter,
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

		deleted, err := vault.Delete(path)
		if err != nil {
			return err
		}
		if !deleted {
			return fmt.Errorf("'%s' not found", path)
		}

		if err := store.Save(vault, password); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Deleted '%s'.\n", path)
		return nil
	},
}
