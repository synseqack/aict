// Package testutil provides small helpers shared by aict's tool test suites.
package testutil

import (
	"path/filepath"
	"testing"
)

// MissingPath returns an absolute path, ending in name, that is guaranteed not
// to exist for the duration of the test.
//
// Tests must not hardcode paths such as "/nonexistent/file.txt". On Windows
// that resolves to an ordinary, creatable location (C:\nonexistent\file.txt)
// which may genuinely exist on a developer's machine — silently inverting the
// assertion, so a "missing path" test starts exercising a real file instead.
// Anchoring to t.TempDir() gives a directory the test framework created empty
// and removes on cleanup, so name is reliably absent on every platform.
func MissingPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), name)
}
