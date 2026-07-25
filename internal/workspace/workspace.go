// Package workspace implements the grove lifecycle: create and finish.
//
// A grove is a directory of git worktrees drawn from several repos, plus a
// manifest snapshot and generated context. The directory is the unit; the
// task is what you do inside it.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jossmoff/grove/internal/config"
	"github.com/jossmoff/grove/internal/contextgen"
	"github.com/jossmoff/grove/internal/gitx"
	"github.com/jossmoff/grove/internal/index"
	"github.com/jossmoff/grove/internal/manifest"
	"github.com/jossmoff/grove/internal/status"
)

// Selection is one resolved repo plus its role and optional pinning, ready to
// materialise.
type Selection struct {
	Entry index.Entry
	Role  manifest.Role
	Base  string // empty → origin/HEAD at creation
	Dir   string // empty → repo leaf
}

// CreateOpts parameterises Create.
type CreateOpts struct {
	Slug    string
	Task    string
	Profile string // provenance only
	// FetchReads updates read repos before detaching. Reads default to fresh
	// because a stale reference hands the agent an interface that no longer
	// exists — a correctness problem. Writes are left alone: branching from
	// what you last pulled is often deliberate.
	FetchReads bool
	ProfileDir string // where CONTEXT.md/LOCAL.md are copied from, if set
	AgentNotes []string
}

// Create materialises a grove. On any failure it rolls back everything it
// created: a half-built grove is worse than none.
func Create(cfg config.Config, sel []Selection, opts CreateOpts) (string, error) {
	if len(sel) == 0 {
		return "", errors.New("no repos selected")
	}
	groveDir := cfg.GroveDir(opts.Slug)
	if _, err := os.Stat(groveDir); err == nil {
		return "", fmt.Errorf("grove %q already exists at %s", opts.Slug, groveDir)
	}
	branch := cfg.BranchFor(opts.Slug)

	// Validate everything before mutating anything.
	seen := map[string]string{}
	for i := range sel {
		s := &sel[i]
		if s.Dir == "" {
			s.Dir = s.Entry.Canonical[strings.LastIndexByte(s.Entry.Canonical, '/')+1:]
		}
		if prev, dup := seen[s.Dir]; dup {
			return "", fmt.Errorf("directory collision: %s and %s both want %q — set dir on one",
				prev, s.Entry.Canonical, s.Dir)
		}
		seen[s.Dir] = s.Entry.Canonical
		if s.Role == manifest.RoleWrite && gitx.BranchExists(s.Entry.Path, branch) {
			return "", fmt.Errorf(
				"branch %s already exists in %s — a previous grove left it behind; delete it or pick another slug",
				branch, s.Entry.Name)
		}
	}

	if opts.FetchReads {
		for _, s := range sel {
			if s.Role == manifest.RoleRead {
				if err := gitx.Fetch(s.Entry.Path); err != nil {
					fmt.Fprintf(os.Stderr, "warning: fetch %s failed — using local refs\n", s.Entry.Name)
				}
			}
		}
	}

	if err := os.MkdirAll(groveDir, 0o755); err != nil {
		return "", err
	}

	type created struct{ repo, wt string }
	var done []created
	rollback := func() {
		for i := len(done) - 1; i >= 0; i-- {
			_ = gitx.WorktreeRemove(done[i].repo, done[i].wt, true)
			_ = gitx.WorktreePrune(done[i].repo)
		}
		_ = os.RemoveAll(groveDir)
	}

	var repos []manifest.Repo
	for _, s := range sel {
		target := filepath.Join(groveDir, s.Dir)
		base := s.Base
		if base == "" {
			base = gitx.DefaultBase(s.Entry.Path)
		}

		var err error
		switch s.Role {
		case manifest.RoleWrite:
			err = gitx.WorktreeAddBranch(s.Entry.Path, target, branch, base)
		case manifest.RoleRead:
			err = gitx.WorktreeAddDetached(s.Entry.Path, target, base)
		default:
			err = fmt.Errorf("unknown role %q", s.Role)
		}
		if err != nil {
			rollback()
			return "", fmt.Errorf("worktree for %s: %w", s.Entry.Name, err)
		}
		done = append(done, created{s.Entry.Path, target})

		repos = append(repos, manifest.Repo{
			At:   s.Entry.Canonical,
			Role: s.Role,
			Dir:  s.Dir,
			Base: s.Base, // as asked for — empty stays empty; git knows the SHA
		})
	}

	m := &manifest.Manifest{
		Slug:    opts.Slug,
		Branch:  branch,
		Created: nowUTC(),
		Task:    opts.Task,
		Profile: opts.Profile,
		Repos:   repos,
	}
	if err := m.Save(groveDir); err != nil {
		rollback()
		return "", err
	}

	if opts.ProfileDir != "" {
		copyNotes(opts.ProfileDir, groveDir)
	}
	if err := contextgen.WriteAll(cfg, groveDir, m); err != nil {
		rollback()
		return "", err
	}
	return groveDir, nil
}

// copyNotes copies CONTEXT.md and LOCAL.md from the profile. Copy, not
// symlink: the profile is the source of truth, and an agent writing through a
// symlink would clobber durable notes.
func copyNotes(profileDir, groveDir string) {
	for _, f := range []string{"CONTEXT.md", "LOCAL.md"} {
		data, err := os.ReadFile(filepath.Join(profileDir, f))
		if err != nil {
			continue
		}
		_ = os.WriteFile(filepath.Join(groveDir, f), data, 0o644)
	}
}

// FinishOpts parameterises Finish.
type FinishOpts struct {
	// Force proceeds despite uncommitted or unpushed work.
	Force bool
}

// Finish tears a grove down: remove each worktree, prune its source repo,
// delete the directory. Branches are deliberately left alone — the work may be
// pushed but unmerged, and deleting them here would silently lose it.
func Finish(cfg config.Config, groveDir string, opts FinishOpts) error {
	m, err := manifest.Load(groveDir)
	if err != nil {
		return err
	}

	if !opts.Force {
		st := status.Grove(groveDir, m)
		var dirty, unpushed []string
		for _, r := range st.Repos {
			if r.Dirty > 0 {
				dirty = append(dirty, fmt.Sprintf("%s (~%d)", r.At, r.Dirty))
			}
			if r.Role == manifest.RoleWrite && r.Ahead > 0 {
				unpushed = append(unpushed, r.At)
			}
		}
		if len(dirty) > 0 {
			return fmt.Errorf("uncommitted changes in %s — commit, or re-run with --force", strings.Join(dirty, ", "))
		}
		if len(unpushed) > 0 {
			return fmt.Errorf("unpushed commits in %s — push, or re-run with --force", strings.Join(unpushed, ", "))
		}
	}

	for _, r := range m.Repos {
		main := filepath.Join(cfg.Root, filepath.FromSlash(r.At))
		wt := filepath.Join(groveDir, r.Dir)
		if _, err := os.Stat(wt); err == nil {
			if err := gitx.WorktreeRemove(main, wt, opts.Force); err != nil {
				return fmt.Errorf("removing worktree %s: %w", wt, err)
			}
		}
		_ = gitx.WorktreePrune(main) // prune only helps once the dir is gone
	}
	return os.RemoveAll(groveDir)
}
