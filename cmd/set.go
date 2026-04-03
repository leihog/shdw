package cmd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/leihog/shdw/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var setForce bool
var setInteractive bool

var setCmd = &cobra.Command{
	Use:   "set <path> [value]",
	Short: "Store a secret",
	Long: `Store a secret at the given path.

Intermediate namespaces are created automatically.
Use --force to overwrite the value of an existing key.
Use --interactive (-i) to enter the value via a hidden prompt, keeping it
out of your shell history.

Errors if the path (or any segment of it) conflicts with an existing node's type.`,
	Example: `  shdw set token abc123
  shdw set discord/api_key abc123
  shdw set discord/prod/token abc123
  shdw set discord/api_key newvalue --force
  shdw set discord/api_key -i`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		var value string
		if setInteractive || len(args) == 1 {
			fmt.Fprintf(os.Stderr, "Value for '%s': ", path)
			raw, err := term.ReadPassword(syscall.Stdin)
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return err
			}
			value = string(raw)
		} else {
			value = args[1]
		}

		password, err := getMasterPassword(true)
		if err != nil {
			return err
		}

		vault, err := openVault(password)
		if err != nil {
			return err
		}

		existed, err := vault.Set(path, value, setForce)
		if err != nil {
			return err
		}

		if err := store.Save(vault, password); err != nil {
			return err
		}

		if existed {
			fmt.Fprintf(os.Stderr, "Updated '%s'.\n", path)
		} else {
			fmt.Fprintf(os.Stderr, "Stored '%s'.\n", path)
		}
		return nil
	},
}

func init() {
	setCmd.Flags().BoolVarP(&setForce, "force", "f", false, "Overwrite an existing key's value")
	setCmd.Flags().BoolVarP(&setInteractive, "interactive", "i", false, "Prompt for value interactively (hidden, keeps value out of shell history)")
}
