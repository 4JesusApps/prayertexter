// Package buildinfo exposes a compact version identifier derived from the
// binary's embedded VCS metadata (populated by `go build` when run inside a
// git working tree). It replaces the historical pattern of injecting a
// version string via -ldflags, which required every build path (Makefile,
// CI, SAM) to opt in or the value silently stayed empty in production.
package buildinfo

import "runtime/debug"

// shortRevLen is the number of leading SHA characters Version() emits — long
// enough to be unambiguous in this repo, short enough to keep log lines tidy.
const shortRevLen = 12

// Version returns a short build identifier: the first shortRevLen characters
// of the commit SHA, with a "-dirty" suffix when the working tree was
// modified at build time. Returns "unknown" when VCS metadata is unavailable
// (e.g. building from a tarball or with -buildvcs=false).
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	return format(info)
}

func format(info *debug.BuildInfo) string {
	var rev string
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return "unknown"
	}
	if len(rev) > shortRevLen {
		rev = rev[:shortRevLen]
	}
	if dirty {
		return rev + "-dirty"
	}
	return rev
}
