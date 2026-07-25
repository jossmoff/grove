// Package status derives grove state. Nothing here is persisted: every field
// is recomputed by asking git, in parallel. If it gets slow, cache with a TTL
// like the index — never write it into the manifest.
package status

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jossmoff/grove/internal/config"
	"github.com/jossmoff/grove/internal/gitx"
	"github.com/jossmoff/grove/internal/manifest"
)

// RepoStatus is the derived state of one repo in a grove.
type RepoStatus struct {
	At      string        `json:"at"`
	Role    manifest.Role `json:"role"`
	Dir     string        `json:"dir"`
	Present bool          `json:"present"`
	Branch  string        `json:"branch,omitempty"`
	Dirty   int           `json:"dirty"`
	Ahead   int           `json:"ahead"`
	Behind  int           `json:"behind"`
}

// Summary renders a one-glance state: "clean", "missing", or "~3 ↑1".
func (r RepoStatus) Summary() string {
	if !r.Present {
		return "missing"
	}
	s := ""
	if r.Dirty > 0 {
		s += fmt.Sprintf("~%d", r.Dirty)
	}
	if r.Ahead > 0 {
		s = join(s, fmt.Sprintf("↑%d", r.Ahead))
	}
	if r.Behind > 0 {
		s = join(s, fmt.Sprintf("↓%d", r.Behind))
	}
	if s == "" {
		return "clean"
	}
	return s
}

func join(a, b string) string {
	if a == "" {
		return b
	}
	return a + " " + b
}

// GroveStatus is the derived state of a whole grove.
type GroveStatus struct {
	Slug    string       `json:"slug"`
	Branch  string       `json:"branch"`
	Task    string       `json:"task,omitempty"`
	Created time.Time    `json:"created"`
	Dir     string       `json:"dir"`
	Repos   []RepoStatus `json:"repos"`
}

// DirtyRepos counts repos with uncommitted changes.
func (g GroveStatus) DirtyRepos() int {
	n := 0
	for _, r := range g.Repos {
		if r.Dirty > 0 {
			n++
		}
	}
	return n
}

// Grove derives status for one grove from its manifest.
func Grove(groveDir string, m *manifest.Manifest) GroveStatus {
	repos := make([]RepoStatus, len(m.Repos))
	var wg sync.WaitGroup
	for i, r := range m.Repos {
		wg.Add(1)
		go func(i int, r manifest.Repo) {
			defer wg.Done()
			p := filepath.Join(groveDir, r.Dir)
			rs := RepoStatus{At: r.At, Role: r.Role, Dir: r.Dir}
			if _, err := os.Stat(p); err != nil {
				repos[i] = rs
				return
			}
			rs.Present = true
			if b, ok := gitx.CurrentBranch(p); ok {
				rs.Branch = b
			}
			rs.Dirty = gitx.DirtyCount(p)
			if a, b, ok := gitx.AheadBehind(p); ok {
				rs.Ahead, rs.Behind = a, b
			}
			repos[i] = rs
		}(i, r)
	}
	wg.Wait()

	return GroveStatus{
		Slug: m.Slug, Branch: m.Branch, Task: m.Task,
		Created: m.Created, Dir: groveDir, Repos: repos,
	}
}

// All discovers every grove under the grove root, newest first.
func All(cfg config.Config) ([]GroveStatus, error) {
	entries, err := os.ReadDir(cfg.GroveRoot)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []GroveStatus
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(cfg.GroveRoot, e.Name())
		m, err := manifest.Load(dir)
		if err != nil {
			continue
		}
		out = append(out, Grove(dir, m))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	return out, nil
}
