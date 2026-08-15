package markdir

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
)

const (
	routeDoc = iota
	routeIndex
)

var errNotFound = errors.New("not found")

// route describes what a request URL resolves to on disk.
type route struct {
	kind     int
	diskPath string // absolute disk path: file for routeDoc, dir for routeIndex
	urlPath  string // canonical URL path of the route: the file for docs, the dir for indexes
}

// server renders a directory of markdown files over HTTP.
type server struct {
	root     string // absolute, cleaned MD_DIR
	realRoot string // EvalSymlinks(root), the containment boundary
	md       goldmark.Markdown
	css      []byte
}

// NewHandler returns an HTTP handler that serves the markdown files under
// root. Every request reads from disk, so edits show up on refresh.
func NewHandler(root string) (http.Handler, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	abs = filepath.Clean(abs)
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", abs, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}
	realRoot, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", abs, err)
	}
	return &server{
		root:     abs,
		realRoot: realRoot,
		md:       newMarkdown(),
		css:      assembleCSS(),
	}, nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// The stylesheet is served from the binary, never from MD_DIR.
	if r.URL.Path == "/styles.css" {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(s.css)
		return
	}
	rt, err := s.resolve(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch rt.kind {
	case routeDoc:
		s.renderDoc(w, r, rt)
	case routeIndex:
		s.renderIndex(w, rt)
	}
}

// resolve maps a URL path to a route. It returns errNotFound for anything
// that does not resolve to a markdown file or directory inside root.
func (s *server) resolve(urlPath string) (route, error) {
	p := path.Clean(urlPath)
	rel := filepath.FromSlash(strings.TrimPrefix(p, "/"))
	disk := filepath.Join(s.root, rel)

	// EvalSymlinks resolves every component; comparing the result against
	// realRoot keeps symlinks from reaching outside MD_DIR.
	real, err := filepath.EvalSymlinks(disk)
	if err != nil {
		return route{}, errNotFound
	}
	if !inside(real, s.realRoot) {
		return route{}, errNotFound
	}
	info, err := os.Stat(real)
	if err != nil {
		return route{}, errNotFound
	}
	if info.IsDir() {
		entries, err := os.ReadDir(real)
		if err != nil {
			return route{}, fmt.Errorf("read %s: %w", real, err)
		}
		if readme, ok := findReadme(entries); ok {
			readmeDisk := filepath.Join(real, readme.Name())
			readmeReal, err := filepath.EvalSymlinks(readmeDisk)
			if err != nil || !inside(readmeReal, s.realRoot) {
				// A README escaping the root via symlink is treated as absent.
				return route{kind: routeIndex, diskPath: real, urlPath: p}, nil
			}
			return route{kind: routeDoc, diskPath: readmeDisk, urlPath: path.Join(p, readme.Name())}, nil
		}
		return route{kind: routeIndex, diskPath: real, urlPath: p}, nil
	}
	if !strings.EqualFold(filepath.Ext(real), ".md") {
		return route{}, errNotFound
	}
	return route{kind: routeDoc, diskPath: real, urlPath: p}, nil
}

// findReadme prefers an exact README.md, falling back to a case-folded match.
func findReadme(entries []fs.DirEntry) (fs.DirEntry, bool) {
	var fold fs.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if e.Name() == "README.md" {
			return e, true
		}
		if fold == nil && strings.EqualFold(e.Name(), "readme.md") {
			fold = e
		}
	}
	if fold != nil {
		return fold, true
	}
	return nil, false
}

// inside reports whether real is realRoot or a descendant of it.
func inside(real, realRoot string) bool {
	return real == realRoot || strings.HasPrefix(real, realRoot+string(filepath.Separator))
}

// urlJoin appends an escaped path segment to a canonical URL path.
func urlJoin(dirURL, name string) string {
	seg := url.PathEscape(name)
	if dirURL == "/" {
		return "/" + seg
	}
	return dirURL + "/" + seg
}
