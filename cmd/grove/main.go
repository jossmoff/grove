// Command grove materialises multi-repo agent workspaces backed by git
// worktrees.
//
// Design rules the surface follows:
//
//   - stdout is the answer, stderr is the narration: each command prints
//     exactly one machine-readable thing (a path, a listing) to stdout.
//   - flat verbs on the primary noun (new, ls, path, sync, finish); nested
//     noun-verb for secondary nouns (profile …, repo …).
//   - [slug] optional everywhere: commands discover the enclosing grove by
//     walking up for grove.toml, the way git walks up for .git.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/jossmoff/grove/internal/cmd"
)

// version is stamped by goreleaser via -ldflags at release time.
var version = "dev"

func main() {
	app := &cli.Command{
		Name:    "grove",
		Version: version,
		Usage:   "multi-repo agent workspaces backed by git worktrees",
		Description: "Grove materialises a directory where several repos sit as siblings,\n" +
			"each a git worktree with a role: write (a branch you'll commit to) or\n" +
			"read (a detached checkout for context). Point your agent at the\n" +
			"directory; grove stays out of the loop.",
		Commands: []*cli.Command{
			cmd.New(),
			cmd.Ls(),
			cmd.Path(),
			cmd.Sync(),
			cmd.Finish(),
			cmd.Profile(),
			cmd.Repo(),
			cmd.Init(),
			cmd.Config(),
		},
		EnableShellCompletion: true,
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "grove:", err)
		os.Exit(1)
	}
}
