package markdir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListDirFilterAndSort(t *testing.T) {
	// Note: no names that differ only by case — case-insensitive filesystems
	// (macOS APFS) would collapse them into one file.
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"ai.md":      "x",
		"ai/x.md":    "x",
		"a.md":       "x",
		"bo.md":      "x",
		"notes.txt":  "x",
		".hidden.md": "x",
		"sub/x.md":   "x",
	})
	realRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := listDir(realRoot, dir, "/")
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
	}
	// Directories (ai/, sub/) and non-markdown files are excluded.
	want := []string{"a.md", "ai.md", "bo.md"}
	if len(names) != len(want) {
		t.Fatalf("entries = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("entries = %v, want %v", names, want)
		}
	}
}

func TestListDirSkipsEscapingSymlink(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"README.md": "# R\n"})
	if err := os.Symlink("/etc", filepath.Join(dir, "escape")); err != nil {
		t.Fatal(err)
	}
	realRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := listDir(realRoot, dir, "/")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name == "escape" {
			t.Error("listDir should skip symlinks escaping the root")
		}
	}
}
