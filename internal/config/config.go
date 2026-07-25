// Package config loads grove's own configuration.
//
// Precedence: $GROVE_CONFIG, else $XDG_CONFIG_HOME/grove/config.toml, else
// ~/.config/grove/config.toml. The env override exists so tests (and unusual
// setups) never need to fake a home directory.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is grove's global configuration. All fields have working defaults;
// an absent file is not an error.
type Config struct {
	// Root is where clones live, ghq-style: <root>/<host>/<owner>/<repo>.
	Root string `toml:"root"`
	// GroveRoot is where groves live: <grove_root>/<slug>/.
	GroveRoot string `toml:"grove_root"`
	// BranchPrefix prefixes write branches: <prefix>/<slug>.
	BranchPrefix string `toml:"branch_prefix"`
	// IndexTTLSecs is how long the repo index cache stays warm.
	IndexTTLSecs int `toml:"index_ttl_secs"`
	// AgentFile is the filename agents read. Configurable because different
	// agent CLIs look for different names; this is the whole integration
	// surface with them.
	AgentFile string `toml:"agent_file"`
	// SkillsDir is the per-repo directory holding agent skills.
	SkillsDir string `toml:"skills_dir"`
}

// Default returns the built-in configuration.
func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Root:         filepath.Join(home, "src"),
		GroveRoot:    filepath.Join(home, "grove"),
		BranchPrefix: "grove",
		IndexTTLSecs: 30,
		AgentFile:    "AGENTS.md",
		SkillsDir:    filepath.Join(".claude", "skills"),
	}
}

// Path returns where config is (or would be) read from.
func Path() string {
	if p := os.Getenv("GROVE_CONFIG"); p != "" {
		return p
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "grove", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "grove", "config.toml")
}

// CacheDir returns where derived data (the repo index) is cached.
func CacheDir() string {
	if p := os.Getenv("GROVE_CACHE_DIR"); p != "" {
		return p
	}
	if x := os.Getenv("XDG_CACHE_HOME"); x != "" {
		return filepath.Join(x, "grove")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "grove")
}

// ProfilesDir returns where profiles live, beside the config file.
func ProfilesDir() string {
	return filepath.Join(filepath.Dir(Path()), "profiles")
}

// Load reads config from Path(). A missing file yields Default(); an unknown
// key is an error, because a typo silently ignored is a config that lies.
func Load() (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(Path())
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	meta, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return cfg, fmt.Errorf("parse %s: %w", Path(), err)
	}
	if und := meta.Undecoded(); len(und) > 0 {
		keys := make([]string, len(und))
		for i, k := range und {
			keys[i] = k.String()
		}
		return cfg, fmt.Errorf("unknown key(s) in %s: %s", Path(), strings.Join(keys, ", "))
	}
	cfg.expand()
	return cfg, nil
}

// expand resolves a leading ~ in path fields, since TOML has no shell.
func (c *Config) expand() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	for _, p := range []*string{&c.Root, &c.GroveRoot} {
		if strings.HasPrefix(*p, "~/") {
			*p = filepath.Join(home, (*p)[2:])
		}
	}
}

// GroveDir returns the directory for a slug.
func (c Config) GroveDir(slug string) string {
	return filepath.Join(c.GroveRoot, slug)
}

// BranchFor returns the write branch for a slug.
func (c Config) BranchFor(slug string) string {
	return c.BranchPrefix + "/" + slug
}
