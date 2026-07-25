// Package manifest defines grove.toml — the per-grove snapshot.
//
// Two rules carry the design:
//
//  1. Derive, don't store. The manifest holds what was asked for (repos,
//     roles, branch, task); everything git already knows (dirty, ahead/behind)
//     is recomputed on read by the status package. Anything persisted that git
//     already knows will drift.
//
//  2. Snapshot, don't reference. The manifest embeds the fully-resolved repo
//     set at creation time. Profiles get edited; teardown must not break
//     because a profile changed three weeks ago. Finish reads the manifest and
//     nothing else.
package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// FileName is the manifest's name inside a grove — also the marker the
// cwd-discovery walk looks for, the way git looks for .git.
const FileName = "grove.toml"

// Role says what a repo is for in this grove.
type Role string

const (
	// RoleWrite gets a worktree on a real branch. You intend to commit.
	RoleWrite Role = "write"
	// RoleRead gets a detached worktree: no branch, nothing to push, nothing
	// left in refs/heads. Context only; edits are discardable by construction.
	RoleRead Role = "read"
)

// Valid reports whether r is a known role.
func (r Role) Valid() bool { return r == RoleWrite || r == RoleRead }

// Repo is one fully-resolved repo entry in the snapshot.
type Repo struct {
	// At is the canonical position under the root: github.com/joss/polywit.
	At   string `toml:"at"`
	Role Role   `toml:"role"`
	// Dir is the directory name inside the grove. Defaults to the repo leaf;
	// stored explicitly because the snapshot is fully resolved.
	Dir string `toml:"dir"`
	// Base is what the worktree starts from — a branch, tag, or SHA.
	// Stored as asked for, not as the SHA it resolved to: git knows the SHA.
	// Empty means origin/HEAD at creation time.
	Base string `toml:"base,omitempty"`
	// DependsOn lists dirs this repo consumes, declared from the dependent —
	// the Terraform/Compose/Cargo direction, because a new consumer should
	// edit its own entry, not its provider's.
	DependsOn []string `toml:"depends_on,omitempty"`
}

// Manifest is the whole of grove.toml.
type Manifest struct {
	Slug    string    `toml:"slug"`
	Branch  string    `toml:"branch"`
	Created time.Time `toml:"created"`
	// Task is the one-line intent. The task is what you're doing; the grove is
	// merely where.
	Task string `toml:"task,omitempty"`
	// Profile records provenance when created from one. Displayed, never
	// acted on — see the snapshot rule.
	Profile string `toml:"profile,omitempty"`
	Repos   []Repo `toml:"repos"`
}

// Writes returns the repos with RoleWrite.
func (m *Manifest) Writes() []Repo { return m.byRole(RoleWrite) }

// Reads returns the repos with RoleRead.
func (m *Manifest) Reads() []Repo { return m.byRole(RoleRead) }

func (m *Manifest) byRole(r Role) []Repo {
	var out []Repo
	for _, e := range m.Repos {
		if e.Role == r {
			out = append(out, e)
		}
	}
	return out
}

// Load reads the manifest from a grove directory.
func Load(groveDir string) (*Manifest, error) {
	p := filepath.Join(groveDir, FileName)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	meta, err := toml.Decode(string(data), &m)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	if und := meta.Undecoded(); len(und) > 0 {
		return nil, fmt.Errorf("unknown key %q in %s", und[0].String(), p)
	}
	return &m, nil
}

// Save writes the manifest into a grove directory.
func (m *Manifest) Save(groveDir string) error {
	if err := os.MkdirAll(groveDir, 0o755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(groveDir, FileName))
	if err != nil {
		return err
	}
	if encErr := toml.NewEncoder(f).Encode(m); encErr != nil {
		_ = f.Close()
		return encErr
	}
	return f.Close()
}

// Discover walks up from dir looking for a grove.toml, the way git walks up
// for .git. Returns the grove directory, or ok=false at the filesystem root.
func Discover(dir string) (string, bool) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, FileName)); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
