package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "set-namespace" {
		fmt.Fprintln(os.Stderr, "usage: demo-kube set-namespace --dir DIR --namespace NAME")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("set-namespace", flag.ContinueOnError)
	dir := fs.String("dir", "", "service directory")
	ns := fs.String("namespace", "", "namespace")
	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	if *dir == "" || *ns == "" {
		fmt.Fprintln(os.Stderr, "set-namespace requires --dir and --namespace")
		os.Exit(2)
	}
	if err := os.WriteFile(filepath.Join(*dir, "namespace.txt"), []byte(*ns+"\n"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
