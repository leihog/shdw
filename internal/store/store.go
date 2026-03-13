package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leihog/shdw/internal/crypto"
	"github.com/leihog/shdw/internal/file"
)

// re-export so callers can check without importing crypto directly
var ErrUnsupportedVersion = crypto.ErrUnsupportedVersion

const vaultFile = "vault"

// NodeType distinguishes keys (leaf nodes) from namespaces (branch nodes).
type NodeType string

const (
	NodeTypeKey       NodeType = "key"
	NodeTypeNamespace NodeType = "namespace"
)

// VaultNode is a node in the vault tree.
// - Namespace nodes have Children and no Value.
// - Key nodes have a Value and no Children.
type VaultNode struct {
	Type     NodeType              `json:"type"`
	Value    string                `json:"value,omitempty"`
	Children map[string]*VaultNode `json:"children,omitempty"`
}

func newNamespace() *VaultNode {
	return &VaultNode{Type: NodeTypeNamespace, Children: make(map[string]*VaultNode)}
}

func newKey(value string) *VaultNode {
	return &VaultNode{Type: NodeTypeKey, Value: value}
}

// Vault wraps the root namespace node.
type Vault struct {
	Root *VaultNode `json:"root"`
}

// Secret is a resolved key with its full path and value.
type Secret struct {
	Path  string // e.g. "discord/prod/token"
	Value string
}

// EnvVarName converts the key name (last segment of Path) to UPPER_SNAKE_CASE.
// With addPathPrefix=true, the full path is used instead.
func (s Secret) EnvVarName(addPathPrefix bool) string {
	if addPathPrefix {
		return toEnvVar(s.Path)
	}
	parts := strings.Split(s.Path, "/")
	return toEnvVar(parts[len(parts)-1])
}

// ── Vault load/save ──────────────────────────────────────────────────────────

func VaultPath() (string, error) {
	dir, err := vaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, vaultFile), nil
}

func Load(password string) (*Vault, error) {
	path, err := VaultPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return newVault(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	plaintext, err := crypto.Decrypt(data, password)
	if err != nil {
		if errors.Is(err, crypto.ErrUnsupportedVersion) {
			return nil, fmt.Errorf("%w\nhint: delete %s and run any 'shdw set' command to create a fresh vault", err, path)
		}
		return nil, err
	}

	var v Vault
	if err := json.Unmarshal(plaintext, &v); err != nil {
		return nil, err
	}
	if v.Root == nil {
		v.Root = newNamespace()
	}
	if v.Root.Children == nil {
		v.Root.Children = make(map[string]*VaultNode)
	}
	return &v, nil
}

func Save(v *Vault, password string) error {
	dir, err := vaultDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	plaintext, err := json.Marshal(v)
	if err != nil {
		return err
	}

	ciphertext, err := crypto.Encrypt(plaintext, password)
	if err != nil {
		return err
	}

	path := filepath.Join(dir, vaultFile)
	if err := backupVaultFile(path); err != nil {
		return fmt.Errorf("backup vault: %w", err)
	}

	return writeVaultFile(path, ciphertext)
}

func backupVaultFile(path string) error {
	old, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return writeVaultFile(path+".bak", old)
}

func writeVaultFile(path string, data []byte) error {
	aw, err := file.NewAtomicWriter(path, 0o600)
	if err != nil {
		return err
	}
	defer aw.Abort()

	if _, err := aw.Write(data); err != nil {
		return err
	}

	return aw.Commit()
}

// ── Path helpers ─────────────────────────────────────────────────────────────

// splitPath splits "discord/prod/token" into ["discord", "prod", "token"].
// Leading/trailing slashes are trimmed. Empty string returns nil.
func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

// walk traverses the tree along segments, returning the node and its parent.
// Returns (nil, nil, nil) if the path doesn't exist.
// Returns an error if a non-terminal segment is a key (can't descend into a key).
func (v *Vault) walk(segments []string) (node *VaultNode, parent *VaultNode, err error) {
	current := v.Root
	var par *VaultNode

	for i, seg := range segments {
		if current.Type == NodeTypeKey {
			return nil, nil, fmt.Errorf(
				"'%s' is a key, not a namespace", strings.Join(segments[:i], "/"),
			)
		}
		child, ok := current.Children[seg]
		if !ok {
			return nil, nil, nil // path doesn't exist
		}
		par = current
		current = child
	}
	return current, par, nil
}

// ── Vault operations ─────────────────────────────────────────────────────────

// Get retrieves a single key by path. Path must point to a key node.
func (v *Vault) Get(path string) (Secret, error) {
	segments := splitPath(path)
	if len(segments) == 0 {
		return Secret{}, fmt.Errorf("path cannot be empty")
	}

	node, _, err := v.walk(segments)
	if err != nil {
		return Secret{}, err
	}
	if node == nil {
		return Secret{}, fmt.Errorf("'%s' not found", path)
	}
	if node.Type == NodeTypeNamespace {
		return Secret{}, fmt.Errorf("'%s' is a namespace — specify a key, e.g. '%s/keyname'", path, path)
	}
	return Secret{Path: path, Value: node.Value}, nil
}

// Set stores a key value at path.
// Returns (existed bool, err).
// Errors if any intermediate segment is a key, or if the target exists as a namespace.
// With force=false, errors if the target key already exists.
func (v *Vault) Set(path, value string, force bool) (existed bool, err error) {
	segments := splitPath(path)
	if len(segments) == 0 {
		return false, fmt.Errorf("path cannot be empty")
	}

	// Walk/create intermediate namespace nodes.
	current := v.Root
	for i, seg := range segments[:len(segments)-1] {
		if current.Type == NodeTypeKey {
			return false, fmt.Errorf(
				"'%s' is a key, not a namespace — delete it first to create a namespace there",
				strings.Join(segments[:i], "/"),
			)
		}
		child, ok := current.Children[seg]
		if !ok {
			// Create intermediate namespace
			child = newNamespace()
			current.Children[seg] = child
		} else if child.Type == NodeTypeKey {
			return false, fmt.Errorf(
				"'%s' is a key, not a namespace — delete it first to create a namespace there",
				strings.Join(segments[:i+1], "/"),
			)
		}
		current = child
	}

	// Now current is the direct parent namespace; handle the final segment.
	finalSeg := segments[len(segments)-1]
	existing, exists := current.Children[finalSeg]

	if exists {
		if existing.Type == NodeTypeNamespace {
			return false, fmt.Errorf(
				"'%s' is a namespace — specify a key inside it, e.g. '%s/keyname'",
				path, path,
			)
		}
		// It's a key.
		if !force {
			return true, fmt.Errorf(
				"'%s' already exists — use --force to overwrite", path,
			)
		}
		current.Children[finalSeg] = newKey(value)
		return true, nil
	}

	current.Children[finalSeg] = newKey(value)
	return false, nil
}

// Delete removes a node (key or namespace) at path.
// Returns false if it didn't exist.
func (v *Vault) Delete(path string) (bool, error) {
	segments := splitPath(path)
	if len(segments) == 0 {
		return false, fmt.Errorf("cannot delete root")
	}

	node, parent, err := v.walk(segments)
	if err != nil {
		return false, err
	}
	if node == nil {
		return false, nil
	}

	delete(parent.Children, segments[len(segments)-1])
	return true, nil
}

// Resolve takes a path and returns matching secrets.
// If the path points to a key, returns that single secret.
// If the path points to a namespace, returns all keys within it (non-recursive).
// Empty path resolves the root namespace.
func (v *Vault) Resolve(path string) ([]Secret, error) {
	segments := splitPath(path)

	var node *VaultNode
	if len(segments) == 0 {
		node = v.Root
	} else {
		var err error
		node, _, err = v.walk(segments)
		if err != nil {
			return nil, err
		}
		if node == nil {
			return nil, fmt.Errorf("'%s' not found", path)
		}
	}

	if node.Type == NodeTypeKey {
		return []Secret{{Path: path, Value: node.Value}}, nil
	}

	// Namespace: return direct child keys only (non-recursive)
	secrets := make([]Secret, 0)
	for _, name := range sortedKeys(node.Children) {
		child := node.Children[name]
		if child.Type == NodeTypeKey {
			childPath := name
			if path != "" {
				childPath = path + "/" + name
			}
			secrets = append(secrets, Secret{Path: childPath, Value: child.Value})
		}
	}
	return secrets, nil
}

// Rename moves a key or namespace from oldPath to newPath.
// Intermediate namespaces at the destination are created automatically.
// Errors if oldPath doesn't exist, newPath is already occupied, or
// any path segment conflicts with an existing node's type.
func (v *Vault) Rename(oldPath, newPath string) error {
	oldSegs := splitPath(oldPath)
	if len(oldSegs) == 0 {
		return fmt.Errorf("cannot rename root")
	}

	// Find the node at oldPath and its parent
	node, parent, err := v.walk(oldSegs)
	if err != nil {
		return err
	}
	if node == nil {
		return fmt.Errorf("'%s' not found", oldPath)
	}

	newSegs := splitPath(newPath)
	if len(newSegs) == 0 {
		return fmt.Errorf("new path cannot be empty")
	}

	// Ensure destination doesn't already exist
	existing, _, err := v.walk(newSegs)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("'%s' already exists — delete it first", newPath)
	}

	// Walk/create intermediate namespaces at destination
	dest := v.Root
	for i, seg := range newSegs[:len(newSegs)-1] {
		if dest.Type == NodeTypeKey {
			return fmt.Errorf("'%s' is a key, not a namespace",
				strings.Join(newSegs[:i], "/"))
		}
		child, ok := dest.Children[seg]
		if !ok {
			child = newNamespace()
			dest.Children[seg] = child
		} else if child.Type == NodeTypeKey {
			return fmt.Errorf("'%s' is a key, not a namespace",
				strings.Join(newSegs[:i+1], "/"))
		}
		dest = child
	}

	finalSeg := newSegs[len(newSegs)-1]
	if dest.Type == NodeTypeKey {
		return fmt.Errorf("cannot move into a key")
	}

	// Move: attach node at new location, remove from old
	dest.Children[finalSeg] = node
	delete(parent.Children, oldSegs[len(oldSegs)-1])

	return nil
}

// NodeAt returns the VaultNode at the given path (empty path = root).
// Returns an error if the path doesn't exist or leads through a key.
func (v *Vault) NodeAt(path string) (*VaultNode, error) {
	segments := splitPath(path)
	if len(segments) == 0 {
		return v.Root, nil
	}
	node, _, err := v.walk(segments)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, fmt.Errorf("'%s' not found", path)
	}
	if node.Type == NodeTypeKey {
		return nil, fmt.Errorf("'%s' is a key, not a namespace", path)
	}
	return node, nil
}

// ResolveMany resolves multiple paths in order, merging results with last-wins
// on env var name collision.
func (v *Vault) ResolveMany(paths []string, addPathPrefix bool) ([]Secret, error) {
	seen := make(map[string]Secret) // envName → secret
	order := []string{}

	for _, path := range paths {
		secrets, err := v.Resolve(path)
		if err != nil {
			return nil, err
		}
		for _, s := range secrets {
			envName := s.EnvVarName(addPathPrefix)
			if _, exists := seen[envName]; !exists {
				order = append(order, envName)
			}
			seen[envName] = s
		}
	}

	result := make([]Secret, 0, len(order))
	for _, name := range order {
		result = append(result, seen[name])
	}
	return result, nil
}

// ── Listing helpers ───────────────────────────────────────────────────────────

// ListChildren returns the names of direct children of the node at path,
// annotated with their type. Used for `shdw list` and shell completion.
type ListEntry struct {
	Name string
	Type NodeType
}

func (v *Vault) ListChildren(path string) ([]ListEntry, error) {
	segments := splitPath(path)

	var node *VaultNode
	if len(segments) == 0 {
		node = v.Root
	} else {
		var err error
		node, _, err = v.walk(segments)
		if err != nil {
			return nil, err
		}
		if node == nil {
			return nil, fmt.Errorf("'%s' not found", path)
		}
		if node.Type == NodeTypeKey {
			return nil, fmt.Errorf("'%s' is a key, not a namespace", path)
		}
	}

	entries := make([]ListEntry, 0, len(node.Children))
	for _, name := range sortedKeys(node.Children) {
		entries = append(entries, ListEntry{Name: name, Type: node.Children[name].Type})
	}
	return entries, nil
}

// AllKeyPaths returns every key path in the vault, for shell completion.
func (v *Vault) AllKeyPaths() []string {
	var paths []string
	walkNode(v.Root, "", &paths)
	sort.Strings(paths)
	return paths
}

// AllNamespacePaths returns every namespace path, for shell completion.
func (v *Vault) AllNamespacePaths() []string {
	var paths []string
	walkNamespaces(v.Root, "", &paths)
	sort.Strings(paths)
	return paths
}

// Stats returns total namespace count and key count across the whole tree.
func (v *Vault) Stats() (namespaces, keys int) {
	countNodes(v.Root, &namespaces, &keys)
	return
}

// ── Internal helpers ─────────────────────────────────────────────────────────

func walkNode(node *VaultNode, prefix string, paths *[]string) {
	for _, name := range sortedKeys(node.Children) {
		child := node.Children[name]
		fullPath := name
		if prefix != "" {
			fullPath = prefix + "/" + name
		}
		if child.Type == NodeTypeKey {
			*paths = append(*paths, fullPath)
		} else {
			walkNode(child, fullPath, paths)
		}
	}
}

func walkNamespaces(node *VaultNode, prefix string, paths *[]string) {
	for _, name := range sortedKeys(node.Children) {
		child := node.Children[name]
		if child.Type != NodeTypeNamespace {
			continue
		}
		fullPath := name
		if prefix != "" {
			fullPath = prefix + "/" + name
		}
		*paths = append(*paths, fullPath)
		walkNamespaces(child, fullPath, paths)
	}
}

func countNodes(node *VaultNode, namespaces, keys *int) {
	for _, child := range node.Children {
		if child.Type == NodeTypeNamespace {
			*namespaces++
			countNodes(child, namespaces, keys)
		} else {
			*keys++
		}
	}
}

func toEnvVar(s string) string {
	s = strings.ToUpper(s)
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

func newVault() *Vault {
	return &Vault{Root: newNamespace()}
}

// SortedChildKeys returns the sorted child names of a node. Exported for use
// in commands that render the tree (e.g. shdw info).
func SortedChildKeys(node *VaultNode) []string {
	return sortedKeys(node.Children)
}

func sortedKeys(m map[string]*VaultNode) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func vaultDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "shdw"), nil
}
