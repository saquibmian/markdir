package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"markdir/internal/markdir"
)

func main() {
	root := os.Getenv("MD_DIR")
	if root == "" {
		fmt.Fprintln(os.Stderr, "markdir: MD_DIR must point to the directory of markdown files to serve")
		os.Exit(2)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	h, err := markdir.NewHandler(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "markdir: %v\n", err)
		os.Exit(1)
	}
	abs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		abs = root
	}
	fmt.Printf("markdir: serving %s at http://localhost:%s\n", abs, port)
	if err := http.ListenAndServe(":"+port, h); err != nil {
		fmt.Fprintf(os.Stderr, "markdir: %v\n", err)
		os.Exit(1)
	}
}
