package cmd

import (
	"fmt"
	"os"

	"github.com/leihog/shdw/internal/store"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show information about the vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultPath, err := store.VaultPath()
		if err != nil {
			return err
		}

		fmt.Printf("  Vault path   %s\n", vaultPath)

		fi, err := os.Stat(vaultPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("  Status       not found (no secrets stored yet)")
				return nil
			}
			return err
		}

		fmt.Printf("  Status       exists\n")
		fmt.Printf("  File size    %s\n", formatBytes(fi.Size()))
		fmt.Printf("  Modified     %s\n", fi.ModTime().Format("2006-01-02 15:04:05"))

		// Use cached password only — don't prompt.
		password, err := getCachedPassword()
		if err != nil || password == "" {
			fmt.Println("  Locked       yes (run 'shdw unlock' to decrypt vault stats)")
			return nil
		}

		vault, err := openVault(password)
		if err != nil {
			fmt.Println("  Locked       yes (cached password invalid)")
			return nil
		}

		namespaces, keys := vault.Stats()
		fmt.Println("  Locked       no")
		fmt.Printf("  Namespaces   %d\n", namespaces)
		fmt.Printf("  Total keys   %d\n", keys)

		return nil
	},
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
