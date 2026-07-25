// Package cmd wires the CLI surface to the internal packages. Each function
// returns one *cli.Command; the mapping from flags to behaviour lives here and
// nowhere else, so the internal packages never see the CLI.
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/jossmoff/grove/internal/config"
	"github.com/jossmoff/grove/internal/contextgen"
	"github.com/jossmoff/grove/internal/index"
	"github.com/jossmoff/grove/internal/manifest"
	"github.com/jossmoff/grove/internal/profile"
	"github.com/jossmoff/grove/internal/status"
	"github.com/jossmoff/grove/internal/tui"
	"github.com/jossmoff/grove/internal/workspace"
)

// groveDirFromArgOrCwd resolves the grove to operate on: an explicit slug if
// given, else cwd discovery. Naming the grove you are standing in is
// backwards, so the argument is optional everywhere.
func groveDirFromArgOrCwd(cfg config.Config, slug string) (string, error) {
	if slug != "" {
		dir := cfg.GroveDir(slug)
		if _, err := os.Stat(filepath.Join(dir, manifest.FileName)); err != nil {
			return "", fmt.Errorf("no grove %q at %s", slug, dir)
		}
		return dir, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir, ok := manifest.Discover(cwd)
	if !ok {
		return "", errors.New("not inside a grove (and no slug given)")
	}
	return dir, nil
}

// New returns the `grove new` command.
func New() *cli.Command {
	return &cli.Command{
		Name:      "new",
		Usage:     "create a grove",
		ArgsUsage: "<slug>",
		Description: "Materialises a grove: one worktree per selected repo, a manifest\n" +
			"snapshot, and generated agent context. Prints the grove's path so\n" +
			"`cd $(grove new x -w polywit)` works.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "profile", Aliases: []string{"p"}, Usage: "create from profile `NAME`"},
			&cli.StringFlag{Name: "task", Aliases: []string{"t"}, Usage: "one-line statement of intent"},
			&cli.StringSliceFlag{Name: "write", Aliases: []string{"w"}, Usage: "add `REPO` as write (repeatable)"},
			&cli.StringSliceFlag{Name: "read", Aliases: []string{"r"}, Usage: "add `REPO` as read (repeatable); REPO may be name@ref to pin"},
			&cli.BoolFlag{Name: "no-fetch", Usage: "skip the default fetch of read repos"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			slug := c.Args().First()
			if slug == "" {
				return errors.New("usage: grove new <slug> [--profile P] [-w REPO]... [-r REPO]...")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			entries, err := index.Load(cfg, false)
			if err != nil {
				return err
			}

			var sel []workspace.Selection
			opts := workspace.CreateOpts{
				Slug:       slug,
				Task:       c.String("task"),
				FetchReads: !c.Bool("no-fetch"),
			}

			if p := c.String("profile"); p != "" {
				pr, err := profile.Load(config.ProfilesDir(), p)
				if err != nil {
					return err
				}
				for _, e := range pr.Repos {
					ent, err := index.Resolve(entries, e.At)
					if err != nil {
						return fmt.Errorf("profile %q: %w", p, err)
					}
					sel = append(sel, workspace.Selection{Entry: ent, Role: e.Role, Base: e.Base, Dir: e.Dir})
				}
				opts.Profile = p
				opts.ProfileDir = profile.Dir(config.ProfilesDir(), p)
			}

			addFlagRepos := func(names []string, role manifest.Role) error {
				for _, n := range names {
					name, base, _ := strings.Cut(n, "@")
					ent, err := index.Resolve(entries, name)
					if err != nil {
						return err
					}
					sel = append(sel, workspace.Selection{Entry: ent, Role: role, Base: base})
				}
				return nil
			}
			if err := addFlagRepos(c.StringSlice("write"), manifest.RoleWrite); err != nil {
				return err
			}
			if err := addFlagRepos(c.StringSlice("read"), manifest.RoleRead); err != nil {
				return err
			}

			// No repos specified any other way → open the interactive picker.
			// This is the "grove new <slug>" with no -w/-r path, and the reason
			// the TUI exists: role selection is ternary, which a flag list
			// makes tedious and a plain multi-select cannot express.
			if len(sel) == 0 {
				picks, err := tui.PickRepos(entries)
				if err != nil {
					return err
				}
				if len(picks) == 0 {
					fmt.Fprintln(os.Stderr, "cancelled")
					return nil
				}
				for _, p := range picks {
					sel = append(sel, workspace.Selection{Entry: p.Entry, Role: p.Role})
				}
			}

			dir, err := workspace.Create(cfg, sel, opts)
			if err != nil {
				return err
			}
			nw, nr := 0, 0
			for _, s := range sel {
				if s.Role == manifest.RoleWrite {
					nw++
				} else {
					nr++
				}
			}
			fmt.Fprintf(os.Stderr, "created %s (%d write, %d read) on %s\n", slug, nw, nr, cfg.BranchFor(slug))
			fmt.Println(dir) // the answer
			return nil
		},
	}
}

// Ls returns the `grove ls` command.
func Ls() *cli.Command {
	return &cli.Command{
		Name:    "ls",
		Aliases: []string{"list"},
		Usage:   "list groves and their derived status",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "emit JSON"},
			&cli.BoolFlag{Name: "interactive", Aliases: []string{"i"}, Usage: "browse groves in an interactive dashboard"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			all, err := status.All(cfg)
			if err != nil {
				return err
			}
			if c.Bool("interactive") {
				return tui.ShowDashboard(all)
			}
			if c.Bool("json") {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(all)
			}
			if len(all) == 0 {
				fmt.Fprintf(os.Stderr, "no groves under %s\n", cfg.GroveRoot)
				return nil
			}
			for _, g := range all {
				flag := "-"
				if g.DirtyRepos() > 0 {
					flag = "*"
				}
				fmt.Printf("%s %-20s %-24s %d repos\n", flag, g.Slug, g.Branch, len(g.Repos))
				for _, r := range g.Repos {
					fmt.Printf("    %-32s %-6s %s\n", r.At, r.Role, r.Summary())
				}
			}
			return nil
		},
	}
}

// Path returns the `grove path` command.
func Path() *cli.Command {
	return &cli.Command{
		Name:      "path",
		Usage:     "print a grove's path",
		ArgsUsage: "[slug]",
		Description: "Prints the path and nothing else. A binary cannot change your\n" +
			"shell's directory; the `gcd` function from `grove init <shell>` wraps\n" +
			"this, the way zoxide's `z` wraps `zoxide query`.",
		Action: func(ctx context.Context, c *cli.Command) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			dir, err := groveDirFromArgOrCwd(cfg, c.Args().First())
			if err != nil {
				return err
			}
			fmt.Println(dir)
			return nil
		},
	}
}

// Sync returns the `grove sync` command.
func Sync() *cli.Command {
	return &cli.Command{
		Name:      "sync",
		Usage:     "regenerate agent context from the manifest",
		ArgsUsage: "[slug]",
		Action: func(ctx context.Context, c *cli.Command) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			dir, err := groveDirFromArgOrCwd(cfg, c.Args().First())
			if err != nil {
				return err
			}
			m, err := manifest.Load(dir)
			if err != nil {
				return err
			}
			if err := contextgen.WriteAll(cfg, dir, m); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "regenerated %s\n", filepath.Join(dir, cfg.AgentFile))
			return nil
		},
	}
}

// Finish returns the `grove finish` command.
func Finish() *cli.Command {
	return &cli.Command{
		Name:      "finish",
		Usage:     "tear a grove down: remove worktrees, prune, delete",
		ArgsUsage: "[slug]",
		Description: "Refuses on uncommitted or unpushed work unless --force. Branches are\n" +
			"never deleted: the work may be pushed but unmerged, and deleting them\n" +
			"here would silently lose it. Prints the parent directory so a shell\n" +
			"wrapper can escape the directory it deleted.",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "proceed despite uncommitted or unpushed work"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			dir, err := groveDirFromArgOrCwd(cfg, c.Args().First())
			if err != nil {
				return err
			}
			if err := workspace.Finish(cfg, dir, workspace.FinishOpts{Force: c.Bool("force")}); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "finished %s\n", filepath.Base(dir))
			fmt.Println(filepath.Dir(dir)) // somewhere that still exists
			return nil
		},
	}
}
