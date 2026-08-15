package markdir

import (
	"strings"
	"testing"

	xhtml "golang.org/x/net/html"
)

func parseDoc(t *testing.T, s string) *xhtml.Node {
	t.Helper()
	doc, err := xhtml.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestExtractTOC(t *testing.T) {
	doc := parseDoc(t, `<h1 id="h1">Top</h1><h2 id="h2-bold">H2 <strong>bold</strong></h2><h3 id="h3-with-code">H3 with <code>code</code></h3><h4>No ID</h4>`)
	toc, first := extractTOC(doc)
	if first != "Top" {
		t.Errorf("firstH1 = %q, want Top", first)
	}
	want := []tocEntry{
		{Level: 1, ID: "h1", Text: "Top"},
		{Level: 2, ID: "h2-bold", Text: "H2 bold"},
		{Level: 3, ID: "h3-with-code", Text: "H3 with code"},
	}
	if len(toc) != len(want) {
		t.Fatalf("len(toc) = %d, want %d (h4 without id excluded): %+v", len(toc), len(want), toc)
	}
	for i, w := range want {
		if toc[i] != w {
			t.Errorf("toc[%d] = %+v, want %+v", i, toc[i], w)
		}
	}
}

func TestExtractTOCNoH1(t *testing.T) {
	doc := parseDoc(t, `<h2 id="a">Only</h2>`)
	toc, first := extractTOC(doc)
	if first != "" {
		t.Errorf("firstH1 = %q, want empty", first)
	}
	if len(toc) != 1 {
		t.Fatalf("len(toc) = %d, want 1", len(toc))
	}
}

func TestExtractTOCSkipsScriptStyle(t *testing.T) {
	doc := parseDoc(t, `<h2 id="x">keep <script>alert(1)</script><style>.s{}</style></h2>`)
	toc, _ := extractTOC(doc)
	if len(toc) != 1 || toc[0].Text != "keep" {
		t.Errorf("toc = %+v, want one entry with text %q", toc, "keep")
	}
}
