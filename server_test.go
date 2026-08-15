package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newFixture builds a Server over a temp dir containing the given files
// (paths relative to the root; parent dirs are created).
func newFixture(t *testing.T, files map[string]string) *Server {
	t.Helper()
	dir := t.TempDir()
	writeFiles(t, dir, files)
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// writeFiles creates files (relative to dir), making parent directories.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func get(t *testing.T, s *Server, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func mustContain(t *testing.T, body string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(body, w) {
			t.Errorf("body missing %q", w)
		}
	}
}

func mustNotContain(t *testing.T, body string, banned ...string) {
	t.Helper()
	for _, b := range banned {
		if strings.Contains(body, b) {
			t.Errorf("body unexpectedly contains %q", b)
		}
	}
}

// specFixture mirrors the example from the spec plus poison entries.
func specFixture(t *testing.T) *Server {
	return newFixture(t, map[string]string{
		"README.md":        "# Root Readme\n\nintro text\n",
		"docs/ai.md":       "# Docs AI\n\ndocs ai doc\n",
		"docs/ai/hello.md": "# Hello\n\nhello world\n",
		"docs/notes.txt":   "not markdown",
		"docs/.hidden.md":  "# Hidden\n",
		"empty/.keep":      "",
	})
}

func TestRoutes(t *testing.T) {
	s := specFixture(t)
	tests := []struct {
		name   string
		target string
		code   int
		inBody []string
		notIn  []string
	}{
		{"root serves README", "/", 200,
			[]string{"Root Readme", `class="tree"`, `class="toc"`, `href="#root-readme"`}, nil},
		{"README.md same as root", "/README.md", 200,
			[]string{"Root Readme", `href="/README.md" class="active"`}, []string{`href="/docs/ai.md"`}},
		{"docs index", "/docs", 200,
			[]string{"ai/", `href="/docs/ai">`, `href="/docs/ai.md"`},
			[]string{"hello.md", "notes.txt", ".hidden.md", `class="tree"`, `class="toc"`}},
		{"docs ai.md doc", "/docs/ai.md", 200,
			[]string{"Docs AI", `href="/docs/ai.md" class="active"`, `href="/docs/ai">`}, nil},
		{"docs ai index", "/docs/ai", 200,
			[]string{"hello.md", `href="/docs/ai/hello.md"`}, []string{"ai.md", `class="tree"`}},
		{"nested doc", "/docs/ai/hello.md", 200,
			[]string{"hello world", `href="/docs/ai/hello.md" class="active"`, `href="/docs/ai.md"`, `href="/docs/ai">`}, nil},
		{"missing", "/nope", 404, nil, nil},
		{"missing md", "/docs/nope.md", 404, nil, nil},
		{"non-md file", "/docs/notes.txt", 404, nil, nil},
		{"extensionless", "/docs/ai/hello", 404, nil, nil},
		{"trailing slash", "/docs/", 200, []string{"ai/", "ai.md"}, nil},
		{"dotdot normalizes", "/docs/../README.md", 200, []string{"Root Readme"}, nil},
		{"dotdot escapes root", "/../../etc/passwd", 404, nil, nil},
		{"encoded dotdot normalizes", "/docs/ai/%2e%2e/ai.md", 200, []string{"Docs AI"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(t, s, tt.target)
			if rec.Code != tt.code {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.code, rec.Body.String())
			}
			mustContain(t, rec.Body.String(), tt.inBody...)
			mustNotContain(t, rec.Body.String(), tt.notIn...)
		})
	}
}

func TestIndexOrder(t *testing.T) {
	s := specFixture(t)
	body := get(t, s, "/docs").Body.String()
	if strings.Index(body, "ai/") > strings.Index(body, "ai.md") {
		t.Errorf("expected ai/ before ai.md in index:\n%s", body)
	}
}

func TestReadmeLowercaseFallback(t *testing.T) {
	// The exact-name preference (README.md over readme.md) is untestable on
	// case-insensitive filesystems, where the two are one file.
	s := newFixture(t, map[string]string{
		"low/readme.md": "# Lower Readme\n",
	})
	body := get(t, s, "/low").Body.String()
	mustContain(t, body, "Lower Readme", `href="/low/readme.md" class="active"`)
}

func TestEmptyDir(t *testing.T) {
	s := specFixture(t)
	body := get(t, s, "/empty").Body.String()
	mustContain(t, body, "This directory is empty.")
}

func TestHiddenFileDirect(t *testing.T) {
	s := specFixture(t)
	body := get(t, s, "/docs/.hidden.md").Body.String()
	mustContain(t, body, "Hidden")
}

func TestRootWithoutReadme(t *testing.T) {
	s := newFixture(t, map[string]string{"docs/ai.md": "# AI\n"})
	body := get(t, s, "/").Body.String()
	mustContain(t, body, "docs/", `href="/docs">`)
	mustNotContain(t, body, `class="tree"`)
}

func TestMethodNotAllowed(t *testing.T) {
	s := specFixture(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "GET, HEAD" {
		t.Errorf("Allow = %q, want %q", got, "GET, HEAD")
	}
}

func TestStyles(t *testing.T) {
	s := specFixture(t)
	rec := get(t, s, "/styles.css")
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Errorf("Content-Type = %q, want text/css", ct)
	}
	mustContain(t, rec.Body.String(), ".markdown-body", ".chroma", ".layout")
}

func TestHTMLHeaders(t *testing.T) {
	s := specFixture(t)
	rec := get(t, s, "/docs/ai.md")
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
}

func TestRefreshSeesUpdates(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.md"), []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, get(t, s, "/hello.md").Body.String(), "v1")

	// Content edits are visible on the next request.
	if err := os.WriteFile(filepath.Join(dir, "hello.md"), []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := get(t, s, "/hello.md").Body.String()
	mustContain(t, body, "v2")
	mustNotContain(t, body, "v1")

	// New directories show up on the next request too.
	writeFiles(t, dir, map[string]string{"new/sub/x.md": "x\n"})
	mustContain(t, get(t, s, "/").Body.String(), "new/")
}

func TestSymlinkSafety(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"README.md":  "# R\n",
		"docs/ai.md": "# A\n",
	})
	if err := os.Symlink("/etc", filepath.Join(dir, "docs", "escape")); err != nil {
		t.Fatal(err)
	}
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	mustNotContain(t, get(t, s, "/docs").Body.String(), "escape")
	if rec := get(t, s, "/docs/escape"); rec.Code != 404 {
		t.Fatalf("escape status = %d, want 404", rec.Code)
	}

	// A loop symlink must not hang the tree or index.
	if err := os.Symlink(".", filepath.Join(dir, "docs", "loop")); err != nil {
		t.Fatal(err)
	}
	mustContain(t, get(t, s, "/docs/ai.md").Body.String(), "ai.md")
	if rec := get(t, s, "/docs"); rec.Code != 200 {
		t.Fatalf("index with loop = %d, want 200", rec.Code)
	}
}

func TestTitleFallback(t *testing.T) {
	s := newFixture(t, map[string]string{"noheading.md": "just text\n"})
	mustContain(t, get(t, s, "/noheading.md").Body.String(), "<title>noheading · markdir</title>")
}

func TestTitleFromH1(t *testing.T) {
	s := newFixture(t, map[string]string{"x.md": "# The Title\n\ntext\n"})
	mustContain(t, get(t, s, "/x.md").Body.String(), "<title>The Title · markdir</title>")
}

func TestRenderFeatures(t *testing.T) {
	s := newFixture(t, map[string]string{"feat.md": strings.Join([]string{
		"| a | b |",
		"| - | - |",
		"| 1 | 2 |",
		"",
		"~~gone~~",
		"",
		"- [ ] todo",
		"",
		`<div class="raw">raw html</div>`,
		"",
		"```go",
		"func main() {}",
		"```",
		"",
	}, "\n")})
	body := get(t, s, "/feat.md").Body.String()
	// Chroma interleaves token spans, so assert the code's pieces, not the
	// contiguous source text.
	mustContain(t, body, "<table>", "<del>", `type="checkbox"`, `<div class="raw">raw html</div>`, "chroma", "func", "main")
}

func TestHrefEscaping(t *testing.T) {
	s := newFixture(t, map[string]string{"docs/a b.md": "# Space\n"})
	mustContain(t, get(t, s, "/docs").Body.String(), `href="/docs/a%20b.md"`)
	mustContain(t, get(t, s, "/docs/a%20b.md").Body.String(), "Space")
}
