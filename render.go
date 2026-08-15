package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	xhtml "golang.org/x/net/html"
)

// newMarkdown builds the goldmark engine. Constructed once in New; goldmark
// is safe for concurrent use.
func newMarkdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithExtensions(highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
			highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
		)),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
	)
}

type DocPage struct {
	Title string
	Body  template.HTML // the rendered markdown; the only unescaped field
	Tree  *Node
	Toc   []TocEntry
}

// renderDoc reads a markdown file from disk (never cached), renders it, and
// serves the doc page with tree and TOC.
func (s *Server) renderDoc(w http.ResponseWriter, r *http.Request, rt route) {
	src, err := os.ReadFile(rt.diskPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r) // file vanished between resolve and read
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var body bytes.Buffer
	if err := s.md.Convert(src, &body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	doc, err := xhtml.Parse(bytes.NewReader(body.Bytes()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toc, firstH1 := extractTOC(doc)
	title := firstH1
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(rt.diskPath), filepath.Ext(rt.diskPath))
	}
	tree, err := buildTree(s.root, s.realRoot, rt.urlPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	page := DocPage{
		Title: title + " · markdir",
		Body:  template.HTML(body.String()),
		Tree:  tree,
		Toc:   toc,
	}
	s.writeHTML(w, s.docTpl, page, rt.diskPath)
}

type IndexPage struct {
	Title   string
	Entries []IndexEntry
}

// renderIndex serves the centered children listing for a directory.
func (s *Server) renderIndex(w http.ResponseWriter, rt route) {
	entries, err := listDir(s.realRoot, rt.diskPath, rt.urlPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	title := strings.TrimPrefix(rt.urlPath, "/")
	if title == "" {
		title = "markdir"
	} else {
		title += " · markdir"
	}
	s.writeHTML(w, s.indexTpl, IndexPage{Title: title, Entries: entries}, rt.diskPath)
}

// writeHTML renders into a buffer first so template errors are reported
// before any bytes hit the wire.
func (s *Server) writeHTML(w http.ResponseWriter, tpl *template.Template, page any, diskPath string) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, page); err != nil {
		log.Printf("render %s: %v", diskPath, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	buf.WriteTo(w)
}
