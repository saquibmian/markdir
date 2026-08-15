package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// IndexEntry is one line of an index page. Name is the on-disk name;
// directories get a trailing "/" at render time.
type IndexEntry struct {
	Name  string
	URL   string
	IsDir bool
}

// listDir returns the visible children of a directory: directories and
// markdown files only, sorted directories-first then case-insensitively.
func listDir(realRoot, dirDisk, dirURL string) ([]IndexEntry, error) {
	entries, err := os.ReadDir(dirDisk)
	if err != nil {
		return nil, err
	}
	out := make([]IndexEntry, 0, len(entries))
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
		isDir := info.IsDir()
		if !isDir && !strings.EqualFold(filepath.Ext(name), ".md") {
			continue
		}
		out = append(out, IndexEntry{Name: name, URL: urlJoin(dirURL, name), IsDir: isDir})
	}
	sort.Slice(out, func(i, j int) bool {
		return nameLess(out[i].Name, out[j].Name, out[i].IsDir, out[j].IsDir)
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
