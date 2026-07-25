package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/BurntSushi/toml"
	"github.com/jossmoff/grove/internal/config"
	"github.com/jossmoff/grove/internal/gitx"
	"github.com/jossmoff/grove/internal/index"
	"github.com/jossmoff/grove/internal/manifest"
	"github.com/jossmoff/grove/internal/profile"
	"github.com/jossmoff/grove/internal/repo"
)

// Profile returns the `grove profile` noun with its verbs.
func Profile() *cli.Command {
	return &cli.Command{
		Name:  "profile",
		Usage: "manage durable repo-set templates",
		Commands: []*cli.Command{
			{
				Name:      "save",
				Usage:     "save the current grove's repo set as a profile",
				ArgsUsage: "<name>",
				Description: "You don't know you want a profile until you've picked the same\n" +
					"repos twice — and by then you're standing in the grove that proves\n" +
					"it. Saves the repo set (roles, bases, dirs); the branch and task\n" +
					"stay with the grove.",
				Action: func(ctx context.Context, c *cli.Command) error {
					name := c.Args().First()
					if name == "" {
						return errors.New("usage: grove profile save <name>")
					}
					cfg, err := config.Load()
					if err != nil {
						return err
					}
					dir, err := groveDirFromArgOrCwd(cfg, "")
					if err != nil {
						return err
					}
					m, err := manifest.Load(dir)
					if err != nil {
						return err
					}
					pr := &profile.Profile{Description: m.Task}
					for _, r := range m.Repos {
						pr.Repos = append(pr.Repos, profile.Entry{
							At: r.At, Role: r.Role, Base: r.Base,
						})
					}
					if err := pr.Save(config.ProfilesDir(), name); err != nil {
						return err
					}
					fmt.Fprintf(os.Stderr, "saved profile %q (%d repos)\n", name, len(pr.Repos))
					fmt.Println(profile.Dir(config.ProfilesDir(), name))
					return nil
				},
			},
			{
				Name:  "ls",
				Usage: "list profiles",
				Action: func(ctx context.Context, c *cli.Command) error {
					names, err := profile.List(config.ProfilesDir())
					if err != nil {
						return err
					}
					for _, n := range names {
						fmt.Println(n)
					}
					return nil
				},
			},
			{
				Name:      "edit",
				Usage:     "open a profile in $EDITOR",
				ArgsUsage: "<name>",
				Action: func(ctx context.Context, c *cli.Command) error {
					name := c.Args().First()
					if name == "" {
						return errors.New("usage: grove profile edit <name>")
					}
					p := filepath.Join(profile.Dir(config.ProfilesDir(), name), "profile.toml")
					if _, err := os.Stat(p); err != nil {
						return fmt.Errorf("no profile %q", name)
					}
					editor := os.Getenv("EDITOR")
					if editor == "" {
						// No editor: print the path, the caller decides.
						fmt.Println(p)
						return nil
					}
					cmd := exec.Command(editor, p)
					cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
					return cmd.Run()
				},
			},
			{
				Name:      "rm",
				Usage:     "delete a profile",
				ArgsUsage: "<name>",
				Action: func(ctx context.Context, c *cli.Command) error {
					name := c.Args().First()
					if name == "" {
						return errors.New("usage: grove profile rm <name>")
					}
					dir := profile.Dir(config.ProfilesDir(), name)
					if _, err := os.Stat(filepath.Join(dir, "profile.toml")); err != nil {
						return fmt.Errorf("no profile %q", name)
					}
					return os.RemoveAll(dir)
				},
			},
			{
				Name:      "export",
				Usage:     "print a profile as one shareable TOML document",
				ArgsUsage: "<name>",
				Description: "Inlines CONTEXT.md; never LOCAL.md — locals hold credentials and\n" +
					"machine paths, and a default that can leak is not a default.",
				Action: func(ctx context.Context, c *cli.Command) error {
					name := c.Args().First()
					if name == "" {
						return errors.New("usage: grove profile export <name>")
					}
					pr, err := profile.Load(config.ProfilesDir(), name)
					if err != nil {
						return err
					}
					doc := struct {
						profile.Profile
						Context string `toml:"context,omitempty"`
					}{Profile: *pr}
					if data, err := os.ReadFile(filepath.Join(profile.Dir(config.ProfilesDir(), name), profile.ContextFile)); err == nil {
						doc.Context = string(data)
					}
					return toml.NewEncoder(os.Stdout).Encode(doc)
				},
			},
			{
				Name:      "import",
				Usage:     "import a profile from an exported TOML file (or - for stdin)",
				ArgsUsage: "<name> <file|->",
				Action: func(ctx context.Context, c *cli.Command) error {
					name, src := c.Args().Get(0), c.Args().Get(1)
					if name == "" || src == "" {
						return errors.New("usage: grove profile import <name> <file|->")
					}
					var data []byte
					var err error
					if src == "-" {
						data, err = readAll(os.Stdin)
					} else {
						data, err = os.ReadFile(src)
					}
					if err != nil {
						return err
					}
					var doc struct {
						profile.Profile
						Context string `toml:"context"`
					}
					if _, err := toml.Decode(string(data), &doc); err != nil {
						return err
					}
					if err := doc.Profile.Save(config.ProfilesDir(), name); err != nil {
						return err
					}
					if doc.Context != "" {
						p := filepath.Join(profile.Dir(config.ProfilesDir(), name), profile.ContextFile)
						if err := os.WriteFile(p, []byte(doc.Context), 0o644); err != nil {
							return err
						}
					}
					fmt.Fprintf(os.Stderr, "imported profile %q (%d repos)\n", name, len(doc.Repos))
					return nil
				},
			},
		},
	}
}

func readAll(f *os.File) ([]byte, error) {
	var buf []byte
	tmp := make([]byte, 4096)
	for {
		n, err := f.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, err
		}
	}
}

// Repo returns the `grove repo` noun with its verbs.
func Repo() *cli.Command {
	return &cli.Command{
		Name:  "repo",
		Usage: "manage the repo collection",
		Commands: []*cli.Command{
			{
				Name:      "clone",
				Usage:     "clone a remote into its canonical position under the root",
				ArgsUsage: "<url>",
				Action: func(ctx context.Context, c *cli.Command) error {
					url := c.Args().First()
					if url == "" {
						return errors.New("usage: grove repo clone <url>")
					}
					cfg, err := config.Load()
					if err != nil {
						return err
					}
					canon, err := repo.Parse(url)
					if err != nil {
						return err
					}
					dest := filepath.Join(cfg.Root, filepath.FromSlash(canon.Rel()))
					if _, err := os.Stat(dest); err == nil {
						return fmt.Errorf("%s already exists at %s", canon.Short(), dest)
					}
					if err := gitx.Clone(url, dest); err != nil {
						return err
					}
					index.Invalidate()
					fmt.Println(dest)
					return nil
				},
			},
			{
				Name:  "ls",
				Usage: "list indexed repos",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "refresh", Usage: "bypass the TTL cache"},
					&cli.BoolFlag{Name: "long", Aliases: []string{"l"}, Usage: "include remotes"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					cfg, err := config.Load()
					if err != nil {
						return err
					}
					entries, err := index.Load(cfg, c.Bool("refresh"))
					if err != nil {
						return err
					}
					for _, e := range entries {
						if c.Bool("long") {
							fmt.Printf("%-44s %s\n", e.Canonical, e.Remote)
						} else {
							fmt.Println(e.Canonical)
						}
					}
					fmt.Fprintf(os.Stderr, "%d repos under %s\n", len(entries), cfg.Root)
					return nil
				},
			},
		},
	}
}

// shellInit is the wrapper installed by `grove init`. It exists because a
// binary cannot cd its parent shell: gcd wraps `grove path`, and gfin escapes
// the directory finish just deleted.
const shellInit = `# grove shell integration
gcd() { cd "$(grove path "$@")" || return; }
gfin() { cd "$(grove finish "$@")" || return; }
`

// Init returns the `grove init` command (zoxide/starship idiom).
func Init() *cli.Command {
	return &cli.Command{
		Name:      "init",
		Usage:     "print shell integration (eval \"$(grove init zsh)\")",
		ArgsUsage: "<bash|zsh|fish>",
		Action: func(ctx context.Context, c *cli.Command) error {
			switch c.Args().First() {
			case "bash", "zsh":
				fmt.Print(shellInit)
			case "fish":
				fmt.Print("function gcd; cd (grove path $argv); end\n" +
					"function gfin; cd (grove finish $argv); end\n")
			default:
				return errors.New("usage: grove init <bash|zsh|fish>")
			}
			return nil
		},
	}
}

// Config returns the `grove config` command.
func Config() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "print the effective configuration",
		Description: "Prints effective config to stdout. Initialise with:\n" +
			"  mkdir -p $(dirname $(grove config --path)) && grove config > $(grove config --path)",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "path", Usage: "print the config file path instead"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Bool("path") {
				fmt.Println(config.Path())
				return nil
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return toml.NewEncoder(os.Stdout).Encode(cfg)
		},
	}
}
