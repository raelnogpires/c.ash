package main

import (
	"fmt"
	"os"

	"c.ash/internal/updater"
)

func main() {
	if err := updater.RunHelper(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
