package cmd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/leihog/shdw/internal/keychain"
	"github.com/leihog/shdw/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// keyCompleter completes full key paths (for get, delete).
func keyCompleter(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	password, err := getMasterPassword(false)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	vault, err := store.Load(password)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return vault.AllKeyPaths(), cobra.ShellCompDirectiveNoFileComp
}

// nsCompleter completes namespace paths (for list, export).
func nsCompleter(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	password, err := getMasterPassword(false)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	vault, err := store.Load(password)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return vault.AllNamespacePaths(), cobra.ShellCompDirectiveNoFileComp
}

// anyCompleter completes both key paths and namespace paths (for run).
func anyCompleter(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	password, err := getMasterPassword(false)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	vault, err := store.Load(password)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	candidates := append(vault.AllNamespacePaths(), vault.AllKeyPaths()...)
	return candidates, cobra.ShellCompDirectiveNoFileComp
}

// getCachedPassword returns the cached master password from the OS keychain
// without prompting. Returns ("", nil) if the vault is locked.
func getCachedPassword() (string, error) {
	return keychain.Get()
}

// getMasterPassword retrieves the master password from the OS keychain,
// or prompts the user if not cached. On first use it asks for confirmation.
func getMasterPassword(confirmIfNew bool) (string, error) {
	cached, err := keychain.Get()
	if err == nil && cached != "" {
		return cached, nil
	}

	vaultPath, err := store.VaultPath()
	if err != nil {
		return "", err
	}

	_, statErr := os.Stat(vaultPath)
	isNew := os.IsNotExist(statErr)

	if isNew && confirmIfNew {
		fmt.Fprintln(os.Stderr, "No vault found. Creating a new Shadow vault.")
		fmt.Fprintln(os.Stderr, "   Choose a master password. You'll need this to access your secrets.")
		fmt.Fprint(os.Stderr, "   Master password: ")
		pw1, err := term.ReadPassword(syscall.Stdin)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		fmt.Fprint(os.Stderr, "   Confirm password: ")
		pw2, err := term.ReadPassword(syscall.Stdin)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		if string(pw1) != string(pw2) {
			return "", fmt.Errorf("passwords do not match")
		}
		password := string(pw1)
		_ = keychain.Set(password)
		return password, nil
	}

	fmt.Fprint(os.Stderr, "Master password: ")
	pw, err := term.ReadPassword(syscall.Stdin)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	password := string(pw)
	_ = keychain.Set(password)
	return password, nil
}
