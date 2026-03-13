package cmd

import (
	"fmt"

	"github.com/leihog/shdw/internal/store"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <path>",
	Short: "Retrieve a single secret (prints value to stdout)",
	Long: `Retrieve a single secret and print its value to stdout.

The value is printed with no trailing newline, making it safe for subshell use:
  export TOKEN=$(shdw get discord/prod/token)`,
	Example: `  shdw get token
  shdw get discord/api_key
  export TOKEN=$(shdw get discord/prod/token)`,
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

		fmt.Print(secret.Value)
		return nil
	},
}
