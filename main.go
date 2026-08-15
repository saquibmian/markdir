package main

import (
	"fmt"
	"net/http"
	"os"
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
	srv, err := New(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "markdir: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("markdir: serving %s at http://localhost:%s\n", srv.root, port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		fmt.Fprintf(os.Stderr, "markdir: %v\n", err)
		os.Exit(1)
	}
}
