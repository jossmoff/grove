// Package index discovers repos under the root.
//
// There is no registry file: the filesystem is the registry. A clone at
// <root>/<host>/<owner>/<repo> is the record of that remote. The cache here is
// TTL'd and invalidatable, so it is wrong for at most index_ttl_secs rather
// than wrong forever.
package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jossmoff/grove/internal/config"
	"github.com/jossmoff/grove/internal/gitx"
)

// Entry is one indexed repo.
type Entry struct {
	// Canonical is the path under root: github.com/joss/polywit.
	Canonical string `json:"canonical"`
	// Name is owner/repo — what listings show.
	Name string `json:"name"`
	// Path is the absolute clone path.
	Path string `json:"path"`
	// LastCommit is epoch seconds of the newest commit; drives recency sort.
	LastCommit int64 `json:"last_commit,omitempty"`
	// Remote is origin's URL, when configured.
	Remote string `json:"remote,omitempty"`
}

type cacheFile struct {
	Generated time.Time `json:"generated"`
	Entries   []Entry   `json:"entries"`
}

const maxDepth = 6

// walk finds git repos under root without descending into them.
func walk(root string) []string {
	var out []string
	var rec func(dir string, depth int)
	rec = func(dir string, depth int) {
		if depth > maxDepth {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			p := filepath.Join(dir, e.Name())
			if gitx.IsRepo(p) {
				out = append(out, p)
				continue // never descend into a repo
			}
			rec(p, depth+1)
		}
	}
	if fi, err := os.Stat(root); err == nil && fi.IsDir() {
		rec(root, 0)
	}
	return out
}

// Build scans the root fresh, ignoring any cache. Per-repo git calls run on a
// bounded worker pool: this is subprocess-bound, so a semaphore of NumCPU-ish
// workers is the right shape.
func Build(cfg config.Config) []Entry {
	paths := walk(cfg.Root)
	entries := make([]Entry, len(paths))

	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			rel, err := filepath.Rel(cfg.Root, p)
			if err != nil {
				return
			}
			canonical := filepath.ToSlash(rel)
			name := canonical
			if _, rest, ok := strings.Cut(canonical, "/"); ok {
				name = rest
			}
			e := Entry{Canonical: canonical, Name: name, Path: p}
			if ts, ok := gitx.LastCommitUnix(p); ok {
				e.LastCommit = ts
			}
			if url, ok := gitx.RemoteURL(p); ok {
				e.Remote = url
			}
			entries[i] = e
		}(i, p)
	}
	wg.Wait()

	// Drop the zero entries from Rel failures, then sort: recent first.
	out := entries[:0]
	for _, e := range entries {
		if e.Canonical != "" {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastCommit != out[j].LastCommit {
			return out[i].LastCommit > out[j].LastCommit
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func cachePath() string {
	return filepath.Join(config.CacheDir(), "index.json")
}

// Load returns the index, from cache when warm.
func Load(cfg config.Config, refresh bool) ([]Entry, error) {
	cp := cachePath()
	if !refresh {
		if data, err := os.ReadFile(cp); err == nil {
			var c cacheFile
			if json.Unmarshal(data, &c) == nil {
				age := time.Since(c.Generated)
				if age >= 0 && age < time.Duration(cfg.IndexTTLSecs)*time.Second {
					return c.Entries, nil
				}
			}
		}
	}

	entries := Build(cfg)
	if data, err := json.MarshalIndent(cacheFile{Generated: time.Now(), Entries: entries}, "", "  "); err == nil {
		_ = os.MkdirAll(filepath.Dir(cp), 0o755)
		_ = os.WriteFile(cp, data, 0o644) // best-effort: a failed cache is a slow next call, not an error
	}
	return entries, nil
}

// Invalidate drops the cache. Called after repo clone.
func Invalidate() {
	_ = os.Remove(cachePath())
}

// Resolve matches a user-supplied name against the index. Accepts a bare leaf
// (polywit), owner/repo, or the full canonical path — whatever is unambiguous.
func Resolve(entries []Entry, needle string) (Entry, error) {
	var hits []Entry
	for _, e := range entries {
		leaf := e.Canonical[strings.LastIndexByte(e.Canonical, '/')+1:]
		if e.Canonical == needle || e.Name == needle || leaf == needle {
			hits = append(hits, e)
		}
	}
	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		return Entry{}, &NotFoundError{Needle: needle}
	default:
		names := make([]string, len(hits))
		for i, h := range hits {
			names[i] = h.Canonical
		}
		return Entry{}, &AmbiguousError{Needle: needle, Matches: names}
	}
}

// NotFoundError reports a needle with no index match.
type NotFoundError struct{ Needle string }

func (e *NotFoundError) Error() string {
	return "no repo matching " + strconv(e.Needle) + " in the index (try `grove repo ls --refresh`)"
}

// AmbiguousError reports a needle with several matches.
type AmbiguousError struct {
	Needle  string
	Matches []string
}

func (e *AmbiguousError) Error() string {
	return strconv(e.Needle) + " is ambiguous: " + strings.Join(e.Matches, ", ")
}

func strconv(s string) string { return "\"" + s + "\"" }
