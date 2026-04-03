package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/leihog/shdw/internal/store"
	"github.com/spf13/cobra"
)

var importNamespace string
var importForce bool

var importCmd = &cobra.Command{
	Use:   "import <.env file>",
	Short: "Import secrets from a .env file",
	Long: `Import KEY=VALUE pairs from a .env file.

Keys are stored at the given namespace path (default: root).
Use --force to overwrite existing keys.`,
	Example: `  shdw import .env
  shdw import .env --namespace discord
  shdw import .env -n discord/prod --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		f, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer f.Close()

		password, err := getMasterPassword(true)
		if err != nil {
			return err
		}

		vault, err := openVault(password)
		if err != nil {
			return err
		}

		added, skipped, overwritten := 0, 0, 0
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

			fullPath := key
			if importNamespace != "" {
				fullPath = importNamespace + "/" + key
			}

			existed, err := vault.Set(fullPath, value, importForce)
			if err != nil {
				// Skip conflicts silently when not forcing, report them
				fmt.Fprintf(os.Stderr, "  skipped '%s': %v\n", fullPath, err)
				skipped++
				continue
			}
			if existed {
				overwritten++
			} else {
				added++
			}
		}

		if err := store.Save(vault, password); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Import complete: %d added, %d overwritten, %d skipped.\n",
			added, overwritten, skipped)
		return nil
	},
}

func init() {
	importCmd.Flags().StringVarP(&importNamespace, "namespace", "n", "", "Target namespace path (default: root)")
	importCmd.Flags().BoolVarP(&importForce, "force", "f", false, "Overwrite existing keys")
}
