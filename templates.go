package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

//go:embed styles.css
var vendoredCSS []byte

// assembleCSS concatenates the vendored github-markdown.css, chroma's class
// CSS for both themes, and the app's layout CSS into the single stylesheet
// served at /styles.css. styles.css is the only file embedded in the binary.
func assembleCSS() []byte {
	var buf bytes.Buffer
	buf.Write(vendoredCSS)
	buf.WriteString("\n\n/* ---- markdir: syntax highlighting (chroma, class-based) ---- */\n")
	f := chromahtml.New(chromahtml.WithClasses(true))
	for _, p := range [][2]string{{"github", "light"}, {"github-dark", "dark"}} {
		fmt.Fprintf(&buf, "@media (prefers-color-scheme: %s) {\n", p[1])
		if err := f.WriteCSS(&buf, styles.Get(p[0])); err != nil {
			panic(err) // bytes.Buffer never fails; styles ship with chroma
		}
		// Override chroma's fixed background with the theme-aware canvas so
		// code blocks match the page in both modes. Must come after WriteCSS.
		fmt.Fprintf(&buf, ".chroma { padding: 12px; background-color: var(--bgColor-muted); }\n")
		buf.WriteString("}\n")
	}
	buf.WriteString(appLayoutCSS)
	return buf.Bytes()
}

var docTpl = template.Must(template.New("doc").Parse(docHTML))
var indexTpl = template.Must(template.New("index").Parse(indexHTML))

const docHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<link rel="stylesheet" href="/styles.css">
</head>
<body>
<div class="layout">
  <aside class="tree"><ul class="root">{{template "treeNode" .Tree}}</ul></aside>
  <main class="markdown-body doc">{{.Body}}</main>
  {{if .Toc}}<nav class="toc">
    <h2>Contents</h2>
    <ul>{{range .Toc}}<li class="toc-h{{.Level}}"><a href="#{{.ID}}">{{.Text}}</a></li>{{end}}</ul>
  </nav>{{end}}
</div>
</body>
</html>
{{define "treeNode"}}<li>{{if .IsDir}}<a href="{{.URL}}">{{.Name}}/</a>{{else}}<a href="{{.URL}}"{{if .Active}} class="active"{{end}}>{{.Name}}</a>{{end}}{{if .Kids}}<ul>{{range .Kids}}{{template "treeNode" .}}{{end}}</ul>{{end}}</li>{{end}}
`

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<link rel="stylesheet" href="/styles.css">
</head>
<body>
<main class="index-wrap">
{{if .Entries}}<ul class="index">{{range .Entries}}<li><a href="{{.URL}}">{{.Name}}{{if .IsDir}}/{{end}}</a></li>{{end}}</ul>
{{else}}<p class="index-empty">This directory is empty.</p>{{end}}
</main>
</body>
</html>
`

const appLayoutCSS = `
/* ---- markdir layout ---- */
/* The vendored stylesheet scopes its color variables to .markdown-body, so
   the layout defines the same Primer palette on :root (identical values)
   for the panes around it. */
:root {
  --bgColor-default: #ffffff;
  --bgColor-muted: #f6f8fa;
  --fgColor-default: #1f2328;
  --fgColor-muted: #59636e;
  --fgColor-accent: #0969da;
  --borderColor-default: #d1d9e0;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bgColor-default: #0d1117;
    --bgColor-muted: #151b23;
    --fgColor-default: #f0f6fc;
    --fgColor-muted: #9198a1;
    --fgColor-accent: #4493f8;
    --borderColor-default: #3d444d;
  }
}

body {
  margin: 0;
  background: var(--bgColor-default);
  color: var(--fgColor-default);
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans", Helvetica, Arial, sans-serif;
}
a { color: var(--fgColor-accent); text-decoration: none; }
a:hover { text-decoration: underline; }

.layout {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr) 240px;
  gap: 32px;
  max-width: 1680px;
  margin: 0 auto;
  padding: 24px;
}

.tree {
  position: sticky;
  top: 0;
  align-self: start;
  max-height: 100vh;
  overflow: auto;
  font-size: 14px;
  padding: 8px 16px 8px 0;
  border-right: 1px solid var(--borderColor-default);
}
.tree ul { list-style: none; margin: 0; padding: 0 0 0 1em; }
.tree ul.root { padding-left: 0; }
.tree li { padding: 2px 0; line-height: 1.5; }
.tree a { color: var(--fgColor-default); }
.tree a.active { font-weight: 600; color: var(--fgColor-accent); }

/* Fill the middle grid track (minmax(0, 1fr)); min-width: 0 lets the doc
   shrink with the track instead of being sized by its widest content. */
.doc { min-width: 0; text-wrap: pretty; padding-bottom: 48px; }

.toc {
  position: sticky;
  top: 0;
  align-self: start;
  max-height: 100vh;
  overflow: auto;
  font-size: 13px;
  padding: 8px 0 8px 16px;
  border-left: 1px solid var(--borderColor-default);
}
.toc h2 { margin: 0 0 8px; font-size: 14px; font-weight: 600; }
.toc ul { list-style: none; margin: 0; padding: 0; }
.toc li { padding: 3px 0; line-height: 1.4; }
.toc li a { color: var(--fgColor-muted); }
.toc li a:hover { color: var(--fgColor-accent); }
.toc .toc-h2 { padding-left: 12px; }
.toc .toc-h3 { padding-left: 24px; }
.toc .toc-h4 { padding-left: 36px; }
.toc .toc-h5 { padding-left: 48px; }
.toc .toc-h6 { padding-left: 60px; }

.index-wrap {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.index { list-style: none; margin: 0; padding: 0; font-size: 18px; }
.index li { padding: 6px 0; text-align: center; }
.index-empty { color: var(--fgColor-muted); }

@media (max-width: 1100px) {
  .layout { grid-template-columns: 1fr; }
  .tree { position: static; max-height: none; border-right: 0; border-bottom: 1px solid var(--borderColor-default); }
  .toc { display: none; }
}
`
