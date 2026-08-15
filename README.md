# markdir

A single-binary markdown browser. Point it at a directory of markdown files
and it serves a browsable website: rendered documents with a file tree on
the left and a table of contents on the right, plus directory indexes.

## Usage

```sh
go build -o .bin/markdir .
MD_DIR=/path/to/docs PORT=8080 .bin/markdir
```

- `MD_DIR` (required) — the directory to serve.
- `PORT` (optional, default `8080`) — the port to listen on.

## Docker

```sh
docker build -t markdir .
docker run -p 8080:8080 -v /path/to/docs:/docs markdir
```

- `MD_DIR` is baked to `/docs` inside the image; mount your docs directory
  there. Files are read from the mounted volume on every request, so edits
  show up on refresh with no container restart.
- `PORT` is still configurable with `-e PORT=…`.

## Routes

For an `MD_DIR` containing:

```
docs/
  ai.md
  ai/
    hello.md
README.md
```

| Route             | Serves                                              |
| ----------------- | --------------------------------------------------- |
| `/`               | `README.md`, rendered (GitHub-style, any dir route serves its own `README.md` if present) |
| `/README.md`      | same as `/`                                          |
| `/docs`           | index page: `ai/`, `ai.md` (immediate children only) |
| `/docs/ai.md`     | `docs/ai.md`, rendered                               |
| `/docs/ai`        | index page: `hello.md`                               |
| `/docs/ai/hello.md` | `docs/ai/hello.md`, rendered                       |

- **Doc pages** show a file tree (expanded only along the path to the current
  file) and a clickable TOC built from headings. The current file is
  highlighted in the tree.
- **Index pages** list only immediate child directories and `.md` files,
  centered. Nothing else renders — text files, images, and binaries 404 even
  on direct request.
- Markdown renders with GitHub-flavored extensions (tables, strikethrough,
  task lists, autolinks) and GitHub-style syntax highlighting for fenced
  code blocks. Raw HTML embedded in markdown is rendered as-is.
- `/styles.css` serves the github-markdown stylesheet plus layout styles,
  embedded in the binary. The page follows the system light/dark preference.

## Behavior notes

- **No caching** — every request reads from disk, so a browser refresh picks
  up edits (HTML responses also send `Cache-Control: no-store`).
- **Directories with a `README.md`** serve it (case-insensitive match,
  exact case preferred). The children index is only shown when no README
  exists. (A `?index=1` escape hatch may come later.)
- **Hidden files** (dotfiles) are excluded from indexes and the tree, but a
  direct request to a hidden `.md` file is served.
- **Symlinks** are followed but may never resolve outside `MD_DIR` — escaped
  targets are hidden from listings and 404 on direct request. Symlink loops
  are terminated by a visited set.
- **Trust** — raw HTML in your files executes in the browser. Only point
  `MD_DIR` at content you trust.

## Development

```sh
go test ./...   # fixtures are built in temp dirs; no setup needed
```

The vendored `styles.css` is
[github-markdown-css](https://github.com/sindresorhus/github-markdown-css)
(MIT, kept with its license header); it is the only file embedded in the
binary. Syntax-highlighting CSS and the layout CSS are generated and
appended in memory at startup.
