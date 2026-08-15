package markdir

import (
	"os"
	"path"
	"path/filepath"
	"testing"
)

func findKid(n *node, name string) *node {
	for _, k := range n.Kids {
		if k.Name == name {
			return k
		}
	}
	return nil
}

func testTree(t *testing.T, files map[string]string, currentURL string) *node {
	t.Helper()
	dir := t.TempDir()
	writeFiles(t, dir, files)
	realRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := buildTree(dir, realRoot, path.Dir(currentURL), currentURL)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestTreeExpandsOnlyCurrentPath(t *testing.T) {
	tree := testTree(t, map[string]string{
		"README.md":        "# R\n",
		"docs/ai.md":       "# A\n",
		"docs/ai/hello.md": "# H\n",
		"docs/other.md":    "# O\n",
		"sub/x.md":         "# X\n",
	}, "/docs/ai/hello.md")

	// Root: docs expanded, sub collapsed, README.md a leaf.
	docs := findKid(tree, "docs")
	if docs == nil || !docs.IsDir {
		t.Fatalf("root should contain docs dir, got %+v", tree.Kids)
	}
	sub := findKid(tree, "sub")
	if sub == nil {
		t.Fatal("root should contain sub dir")
	}
	if sub.Kids != nil {
		t.Errorf("sub should be collapsed, has kids: %+v", sub.Kids)
	}

	// docs: expanded, showing both ai/ and ai.md.
	ai := findKid(docs, "ai")
	if ai == nil || ai.Kids == nil {
		t.Fatalf("docs/ai should be expanded, got %+v", ai)
	}
	if findKid(docs, "ai.md") == nil || findKid(docs, "other.md") == nil {
		t.Errorf("docs should list its files: %+v", docs.Kids)
	}

	// ai: expanded to hello.md.
	if findKid(ai, "hello.md") == nil {
		t.Errorf("ai should list hello.md: %+v", ai.Kids)
	}

	// Sibling dirs of the current file's ancestors stay collapsed.
	if sub.Kids != nil {
		t.Errorf("sub collapsed")
	}
}

func TestTreeCollapsesWhenNotOnPath(t *testing.T) {
	tree := testTree(t, map[string]string{
		"README.md":        "# R\n",
		"docs/ai/hello.md": "# H\n",
	}, "/README.md")

	docs := findKid(tree, "docs")
	if docs == nil {
		t.Fatal("root should contain docs dir")
	}
	if docs.Kids != nil {
		t.Errorf("docs should be collapsed for /README.md, has kids: %+v", docs.Kids)
	}
	if findKid(tree, "README.md") == nil {
		t.Error("root should list README.md")
	}
}

func TestTreeSortsDirsFirst(t *testing.T) {
	tree := testTree(t, map[string]string{
		"README.md": "# R\n",
		"ai.md":     "# A\n",
		"ai/x.md":   "# X\n",
	}, "/README.md")
	want := []string{"ai", "ai.md", "README.md"}
	if len(tree.Kids) != len(want) {
		t.Fatalf("root kids = %+v, want %v", tree.Kids, want)
	}
	for i, w := range want {
		if tree.Kids[i].Name != w {
			t.Errorf("root kids = %+v, want %v", tree.Kids, want)
		}
	}
}

func TestTreeSkipsHiddenAndNonMarkdown(t *testing.T) {
	tree := testTree(t, map[string]string{
		"README.md":   "# R\n",
		".hidden.md":  "# H\n",
		"notes.txt":   "x",
		".hiddendir/": "x",
	}, "/README.md")
	for _, k := range tree.Kids {
		if k.Name[0] == '.' || (!k.IsDir && filepath.Ext(k.Name) != ".md") {
			t.Errorf("unexpected tree entry: %+v", k)
		}
	}
}

func TestTreeSkipsEscapingSymlink(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"README.md": "# R\n"})
	if err := os.Symlink("/etc", filepath.Join(dir, "escape")); err != nil {
		t.Fatal(err)
	}
	realRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := buildTree(dir, realRoot, "/", "/README.md")
	if err != nil {
		t.Fatal(err)
	}
	if findKid(tree, "escape") != nil {
		t.Error("tree should skip symlinks escaping the root")
	}
}
