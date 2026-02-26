package main

import (
	"fmt"
	"os"

	"github.com/ATofighi/mbview-go/internal/mbview"
)

func main() {
	opts, err := mbview.ParseOptions(os.Args[1:], os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if opts.ShowVersion {
		fmt.Println(mbview.Version)
		return
	}
	if opts.ShowHelp {
		fmt.Print(mbview.HelpText())
		return
	}

	if err := mbview.Serve(opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
