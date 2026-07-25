// Package profile defines durable repo-set templates.
//
// A profile is a directory:
//
//	profiles/verification/
//	├── profile.toml   the repo set — versioned, exportable
//	├── CONTEXT.md     durable notes — exported with the profile
//	└── LOCAL.md       machine/personal notes — never exported
//
// The CONTEXT/LOCAL split exists because export forces it: durable notes mix
// shareable structure ("changes flow shared → api → web") with unshareable
// locals (key paths, credentials). Two files mean the default cannot leak.
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/jossmoff/grove/internal/manifest"
)

// Version is the current profile schema version. Profiles are durable and
// hand-edited, so they carry a version; manifests are ephemeral and do not.
const Version = 1

// ContextFile and LocalFile are the two notes files. Both are routed into the
// generated agent doc; only ContextFile is ever exported.
const (
	ContextFile = "CONTEXT.md"
	LocalFile   = "LOCAL.md"
)

// Entry is one repo in a profile. In TOML it may be shorthand or longhand:
//
//	repos = [
//	  "byol:read",                                  # shorthand
//	  { at = "github.com/joss/polywit", role = "write" },
//	]
//
// Shorthand is NAME[:ROLE], role defaulting to read — because reference
// context is the dominant case.
type Entry struct {
	At        string        `toml:"at"`
	Role      manifest.Role `toml:"role"`
	Dir       string        `toml:"dir,omitempty"`
	Base      string        `toml:"base,omitempty"`
	Subtree   string        `toml:"subtree,omitempty"`
	DependsOn []string      `toml:"depends_on,omitempty"`
}

// UnmarshalTOML implements shorthand/longhand decoding (the cargo-dependency
// pattern: terse by default, expands when you need it).
func (e *Entry) UnmarshalTOML(v any) error {
	switch t := v.(type) {
	case string:
		name, role, ok := strings.Cut(t, ":")
		e.At = strings.TrimSpace(name)
		if !ok {
			e.Role = manifest.RoleRead
			return nil
		}
		e.Role = manifest.Role(strings.TrimSpace(role))
		if !e.Role.Valid() {
			return fmt.Errorf("repo %q: unknown role %q (want read or write)", e.At, role)
		}
		return nil
	case map[string]any:
		return decodeTable(t, e)
	default:
		return fmt.Errorf("repo entry must be a string or a table, got %T", v)
	}
}

func decodeTable(m map[string]any, e *Entry) error {
	get := func(k string) (string, error) {
		v, ok := m[k]
		if !ok {
			return "", nil
		}
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("repo key %q must be a string", k)
		}
		delete(m, k)
		return s, nil
	}
	var err error
	if e.At, err = get("at"); err != nil {
		return err
	}
	var role string
	if role, err = get("role"); err != nil {
		return err
	}
	if role == "" {
		e.Role = manifest.RoleRead
	} else {
		e.Role = manifest.Role(role)
	}
	if !e.Role.Valid() {
		return fmt.Errorf("repo %q: unknown role %q", e.At, role)
	}
	if e.Dir, err = get("dir"); err != nil {
		return err
	}
	if e.Base, err = get("base"); err != nil {
		return err
	}
	if e.Subtree, err = get("subtree"); err != nil {
		return err
	}
	if deps, ok := m["depends_on"]; ok {
		list, ok := deps.([]any)
		if !ok {
			return fmt.Errorf("repo %q: depends_on must be an array", e.At)
		}
		for _, d := range list {
			s, ok := d.(string)
			if !ok {
				return fmt.Errorf("repo %q: depends_on entries must be strings", e.At)
			}
			e.DependsOn = append(e.DependsOn, s)
		}
		delete(m, "depends_on")
	}
	if e.At == "" {
		return fmt.Errorf("repo entry missing required key %q", "at")
	}
	for k := range m {
		return fmt.Errorf("repo %q: unknown key %q", e.At, k)
	}
	return nil
}

// Profile is profile.toml.
type Profile struct {
	Version     int     `toml:"version"`
	Description string  `toml:"description,omitempty"`
	Repos       []Entry `toml:"repos"`
}

// Dir returns the directory of a named profile under profilesRoot.
func Dir(profilesRoot, name string) string {
	return filepath.Join(profilesRoot, name)
}

// Load reads a named profile.
func Load(profilesRoot, name string) (*Profile, error) {
	p := filepath.Join(Dir(profilesRoot, name), "profile.toml")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("profile %q: %w", name, err)
	}
	var pr Profile
	if _, err := toml.Decode(string(data), &pr); err != nil {
		return nil, fmt.Errorf("profile %q: %w", name, err)
	}
	if pr.Version != Version {
		return nil, fmt.Errorf("profile %q is schema version %d; this grove understands %d",
			name, pr.Version, Version)
	}
	for i, e := range pr.Repos {
		if e.At == "" {
			return nil, fmt.Errorf("profile %q: repo %d has no name", name, i)
		}
	}
	return &pr, nil
}

// Save writes a profile and creates its notes files if absent. Notes are
// yours, not the tool's: they are never overwritten.
func (pr *Profile) Save(profilesRoot, name string) error {
	dir := Dir(profilesRoot, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	pr.Version = Version
	f, err := os.Create(filepath.Join(dir, "profile.toml"))
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(pr); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	for file, header := range map[string]string{
		ContextFile: "# Context: " + name + "\n\nDurable notes about this repo combination. Exported with the profile.\n",
		LocalFile:   "# Local: " + name + "\n\nMachine-specific paths, credentials, quirks. Never exported.\n",
	} {
		p := filepath.Join(dir, file)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if err := os.WriteFile(p, []byte(header), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// List returns profile names under profilesRoot, sorted by ReadDir order.
func List(profilesRoot string) ([]string, error) {
	entries, err := os.ReadDir(profilesRoot)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			if _, err := os.Stat(filepath.Join(profilesRoot, e.Name(), "profile.toml")); err == nil {
				out = append(out, e.Name())
			}
		}
	}
	return out, nil
}
