package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <path> [path...] -- <command> [args...]",
	Short: "Run a command with secrets injected as environment variables",
	Long: `Spawn a subprocess with secrets injected as environment variables.

Each path before -- can be a namespace (injects all direct child keys)
or a specific key path (injects just that key).

Paths are resolved in order — later paths override earlier ones on env var
name collision, so order matters.

Secrets exist only for the duration of the subprocess.`,
	Example: `  shdw run token -- node app.js
  shdw run discord/api_key -- node app.js
  shdw run discord -- node app.js
  shdw run discord discord/prod -- node app.js
  shdw run discord --add-path-prefix -- node app.js`,
	DisableFlagParsing: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		for _, a := range args {
			if a == "--" {
				return nil, cobra.ShellCompDirectiveDefault
			}
		}
		return anyCompleter(cmd, args, toComplete)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, commandArgs, addPathPrefix, err := parseRunArgs(args)
		if err != nil {
			return err
		}
		if len(paths) == 0 {
			return fmt.Errorf("specify at least one path before --")
		}
		if len(commandArgs) == 0 {
			return fmt.Errorf("specify a command to run after --")
		}

		password, err := getMasterPassword(false)
		if err != nil {
			return err
		}

		vault, err := openVault(password)
		if err != nil {
			return err
		}

		secrets, err := vault.ResolveMany(paths, addPathPrefix)
		if err != nil {
			return err
		}
		if len(secrets) == 0 {
			return fmt.Errorf("no secrets found for the specified path(s)")
		}

		env := os.Environ()
		for _, s := range secrets {
			env = append(env, fmt.Sprintf("%s=%s", s.EnvVarName(addPathPrefix), s.Value))
		}

		c := exec.Command(commandArgs[0], commandArgs[1:]...)
		c.Env = env
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			return err
		}
		return nil
	},
}

// parseRunArgs splits args around -- and extracts --add-path-prefix manually
// (DisableFlagParsing is active so cobra won't do it for us).
func parseRunArgs(args []string) (paths, commandArgs []string, addPathPrefix bool, err error) {
	separatorIdx := -1
	for i, a := range args {
		if a == "--" {
			separatorIdx = i
			break
		}
		if a == "-h" || a == "--help" {
			err = fmt.Errorf("use: shdw run <path> [path...] -- <command> [args...]")
			return
		}
	}

	if separatorIdx < 0 {
		err = fmt.Errorf("missing '--' separator\nUsage: shdw run <path> [path...] -- <command> [args...]")
		return
	}

	before := args[:separatorIdx]
	commandArgs = args[separatorIdx+1:]

	for _, a := range before {
		switch {
		case a == "--add-path-prefix":
			addPathPrefix = true
		case strings.HasPrefix(a, "-"):
			err = fmt.Errorf("unknown flag '%s'", a)
			return
		default:
			paths = append(paths, a)
		}
	}
	return
}
