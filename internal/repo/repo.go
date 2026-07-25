// Package repo maps git remote URLs to canonical host/owner/repo paths.
//
// The tricky case is scp-style: `git@github.com:org/repo.git` is not a URL and
// net/url will not parse it. In Rust this justified the gix-url dependency; in
// Go we handle the scp form explicitly, which is what go-git's transport layer
// does internally too.
package repo

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// Canonical is a repo's position under the root: Host/Owner/Name.
// Owner may contain slashes (GitLab nested groups).
type Canonical struct {
	Host  string
	Owner string
	Name  string
}

// Rel returns the path relative to the repo root, e.g. github.com/joss/polywit.
func (c Canonical) Rel() string {
	return path.Join(c.Host, c.Owner, c.Name)
}

// Short returns owner/name — what pickers and listings show.
func (c Canonical) Short() string {
	return c.Owner + "/" + c.Name
}

// scpRE matches user@host:path — no scheme, single colon, non-numeric path
// start (a numeric start would be ssh://host:2222 misread).
var scpRE = regexp.MustCompile(`^(?:([^@/]+)@)?([^:/]+):([^0-9/][^:]*)$`)

// Parse maps any common git remote form to a Canonical.
//
// Supported: https://, ssh:// (with port), git://, and scp-style
// user@host:path. The path's last segment is the repo name; everything before
// it is the owner, preserving nested GitLab groups.
func Parse(remote string) (Canonical, error) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return Canonical{}, fmt.Errorf("empty remote")
	}

	var host, p string
	if m := scpRE.FindStringSubmatch(remote); m != nil && !strings.Contains(remote, "://") {
		host, p = m[2], m[3]
	} else {
		u, err := url.Parse(remote)
		if err != nil {
			return Canonical{}, fmt.Errorf("parse remote %q: %w", remote, err)
		}
		if u.Host == "" {
			return Canonical{}, fmt.Errorf("remote %q has no host (local paths have no canonical position)", remote)
		}
		host = u.Hostname() // strips :port
		p = u.Path
	}

	p = strings.Trim(p, "/")
	p = strings.TrimSuffix(p, ".git")
	i := strings.LastIndexByte(p, '/')
	if i <= 0 || i == len(p)-1 {
		return Canonical{}, fmt.Errorf("remote %q has no owner/repo path (got %q)", remote, p)
	}
	return Canonical{Host: host, Owner: p[:i], Name: p[i+1:]}, nil
}
