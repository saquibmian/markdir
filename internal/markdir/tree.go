package markdir

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// node is one entry in the file tree. Kids is nil when a directory is
// collapsed (not on the current file's path) or has no visible children.
// Active marks the file being viewed.
type node struct {
	Name   string
	URL    string
	IsDir  bool
	Active bool
	Kids   []*node
}

// buildTree returns the file tree for a page: the root and the directories
// on the path to currentDir are expanded, and the entry matching currentURL
// is Active — the current file on doc pages, the current directory on index
// pages. Directories below currentDir stay collapsed.
func buildTree(root, realRoot, currentDir, currentURL string) (*node, error) {
	visited := map[string]bool{realRoot: true}
	kids, err := walkTree(root, "/", currentDir, currentURL, realRoot, visited)
	if err != nil {
		return nil, err
	}
	return &node{Name: filepath.Base(root), URL: "/", IsDir: true, Active: currentURL == "/", Kids: kids}, nil
}

// walkTree lists dirDisk, recursing only into expanded directories. The
// visited set (keyed on resolved real paths) terminates symlink loops.
func walkTree(dirDisk, dirURL, currentDir, currentURL, realRoot string, visited map[string]bool) ([]*node, error) {
	entries, err := os.ReadDir(dirDisk)
	if err != nil {
		return nil, err
	}
	nodes := make([]*node, 0, len(entries))
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
		nodeURL := urlJoin(dirURL, name)
		node := &node{Name: name, URL: nodeURL, IsDir: isDir, Active: nodeURL == currentURL}
		if isDir {
			expanded := nodeURL == currentDir || strings.HasPrefix(currentDir, nodeURL+"/")
			if expanded && !visited[real] {
				visited[real] = true
				kids, err := walkTree(full, nodeURL, currentDir, currentURL, realRoot, visited)
				if err != nil {
					return nil, err
				}
				node.Kids = kids
			}
		}
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nameLess(nodes[i].Name, nodes[j].Name, nodes[i].IsDir, nodes[j].IsDir)
	})
	return nodes, nil
}
