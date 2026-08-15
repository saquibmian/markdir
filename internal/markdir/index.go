package markdir

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// indexEntry is one line of an index page: a markdown file directly inside
// the directory. Subdirectories never appear here — they are navigated via
// the tree.
type indexEntry struct {
	Name string
	URL  string
}

// listDir returns the visible markdown files directly inside dirDisk,
// sorted case-insensitively.
func listDir(realRoot, dirDisk, dirURL string) ([]indexEntry, error) {
	entries, err := os.ReadDir(dirDisk)
	if err != nil {
		return nil, err
	}
	out := make([]indexEntry, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(dirDisk, name)
		real, err := filepath.EvalSymlinks(full)
		if err != nil || !inside(real, realRoot) {
			continue // dangling symlink, or escapes the root
		}
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		if info.IsDir() || !strings.EqualFold(filepath.Ext(name), ".md") {
			continue
		}
		out = append(out, indexEntry{Name: name, URL: urlJoin(dirURL, name)})
	}
	sort.Slice(out, func(i, j int) bool {
		return nameLess(out[i].Name, out[j].Name, false, false)
	})
	return out, nil
}

// nameLess orders directories before files, then case-insensitive name, then
// exact name. Directories-first is what puts "ai/" ahead of "ai.md".
func nameLess(an, bn string, ad, bd bool) bool {
	if ad != bd {
		return ad
	}
	la, lb := strings.ToLower(an), strings.ToLower(bn)
	if la != lb {
		return la < lb
	}
	return an < bn
}
