package cmd

import (
	"fmt"
	"os"

	"github.com/leihog/shdw/internal/store"
	"github.com/spf13/cobra"
)

const defaultKeyLimit = 5

var listAll bool

var listCmd = &cobra.Command{
	Use:   "list [path]",
	Short: "List vault contents as a tree",
	Long: `List vault contents as a tree, starting from the given path (default: root).

Namespaces are shown with a trailing /, keys without.
Without a path, keys are truncated to 5 per namespace — use --all to show
everything, or specify a path to expand that namespace fully.`,
	Example: `  shdw list
  shdw list discord
  shdw list discord/prod
  shdw list --all`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: nsCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := ""
		if len(args) > 0 {
			targetPath = args[0]
		}

		password, err := getMasterPassword(false)
		if err != nil {
			return err
		}

		vault, err := store.Load(password)
		if err != nil {
			return err
		}

		node, err := vault.NodeAt(targetPath)
		if err != nil {
			return err
		}

		if len(node.Children) == 0 {
			if targetPath == "" {
				fmt.Fprintln(os.Stderr, "Vault is empty.")
			} else {
				fmt.Fprintf(os.Stderr, "'%s' is empty.\n", targetPath)
			}
			return nil
		}

		label := "/"
		if targetPath != "" {
			label = targetPath + "/"
		}
		fmt.Println(label)

		// Direct children of the requested path are always shown in full.
		// Truncation applies only to children of nested namespaces.
		printTree(node, "  ", true, listAll, targetPath)

		ns, keys := countSubtree(node)
		fmt.Printf("\n  %d namespace(s), %d key(s)\n", ns, keys)
		return nil
	},
}

// printTree renders a node's children as an indented tree.
// isRoot=true means direct children are shown in full (no truncation).
// Nested namespaces always truncate unless showAll is set.
// nodePath is the full path of node (empty string = root), used in truncation hints.
func printTree(node *store.VaultNode, indent string, isRoot bool, showAll bool, nodePath string) {
	names := store.SortedChildKeys(node)

	// Separate namespaces and keys so namespaces always appear first
	var namespaces, keys []string
	for _, name := range names {
		if node.Children[name].Type == store.NodeTypeNamespace {
			namespaces = append(namespaces, name)
		} else {
			keys = append(keys, name)
		}
	}

	// Namespaces are always listed, but their contents truncate unless showAll
	for _, name := range namespaces {
		child := node.Children[name]
		fmt.Printf("%s%s/\n", indent, name)
		childPath := name
		if nodePath != "" {
			childPath = nodePath + "/" + name
		}
		printTree(child, indent+"  ", showAll, showAll, childPath)
	}

	// Keys: show all if root level or showAll, otherwise truncate
	limit := defaultKeyLimit
	if isRoot || showAll {
		limit = 0
	}
	shown := 0
	for _, name := range keys {
		if limit > 0 && shown >= limit {
			remaining := len(keys) - shown
			fmt.Printf("%s… %d more (use --all or 'shdw list %s')\n", indent, remaining, nodePath)
			break
		}
		fmt.Printf("%s%s\n", indent, name)
		shown++
	}
}

func countSubtree(node *store.VaultNode) (namespaces, keys int) {
	for _, name := range store.SortedChildKeys(node) {
		child := node.Children[name]
		if child.Type == store.NodeTypeNamespace {
			namespaces++
			ns, k := countSubtree(child)
			namespaces += ns
			keys += k
		} else {
			keys++
		}
	}
	return
}

func init() {
	listCmd.Flags().BoolVar(&listAll, "all", false, "Show all keys without truncation")
}
