package main

import (
	"strings"

	xhtml "golang.org/x/net/html"
)

type TocEntry struct {
	Level int
	ID    string
	Text  string
}

// extractTOC walks the parsed document collecting h1-h6 headings that carry
// an id (headings without one can't be linked), plus the text of the first
// h1 ("" if none).
func extractTOC(doc *xhtml.Node) ([]TocEntry, string) {
	var toc []TocEntry
	firstH1 := ""
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode && isHeading(n.Data) {
			id := ""
			for _, a := range n.Attr {
				if a.Key == "id" {
					id = a.Val
				}
			}
			text := strings.TrimSpace(nodeText(n))
			if id != "" {
				toc = append(toc, TocEntry{Level: int(n.Data[1] - '0'), ID: id, Text: text})
			}
			if n.Data == "h1" && firstH1 == "" {
				firstH1 = text
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return toc, firstH1
}

func isHeading(s string) bool {
	return len(s) == 2 && s[0] == 'h' && s[1] >= '1' && s[1] <= '6'
}

// nodeText concatenates the text content of n, skipping script/style.
func nodeText(n *xhtml.Node) string {
	var b strings.Builder
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.TextNode {
			b.WriteString(n.Data)
			return
		}
		if n.Type == xhtml.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}
