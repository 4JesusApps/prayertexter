package buildinfo //nolint:testpackage // testing unexported format helper

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormat_UnknownWhenNoRevision(t *testing.T) {
	require.Equal(t, "unknown", format(&debug.BuildInfo{}))
}

func TestFormat_CleanRevisionTruncatedTo12(t *testing.T) {
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789deadbeef"},
		},
	}
	require.Equal(t, "abcdef012345", format(info))
}

func TestFormat_DirtySuffix(t *testing.T) {
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789"},
			{Key: "vcs.modified", Value: "true"},
		},
	}
	require.Equal(t, "abcdef012345-dirty", format(info))
}

func TestFormat_ModifiedFalseHasNoSuffix(t *testing.T) {
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789"},
			{Key: "vcs.modified", Value: "false"},
		},
	}
	require.Equal(t, "abcdef012345", format(info))
}

// TestFormat_ShortRevisionNotTruncated guards against an off-by-one panic if
// the SHA is shorter than 12 chars (unlikely in practice but cheap to cover).
func TestFormat_ShortRevisionNotTruncated(t *testing.T) {
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc123"},
		},
	}
	require.Equal(t, "abc123", format(info))
}

// TestVersion_IsNonEmpty smoke-tests the live path: the test binary is built
// from a git tree so debug.ReadBuildInfo should return either a real SHA or
// the documented "unknown" sentinel — never the empty string.
func TestVersion_IsNonEmpty(t *testing.T) {
	require.NotEmpty(t, Version())
}
