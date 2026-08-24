// Command macosbundle wraps a giffer binary as Giffer.app for double-click launch.
// Usage: go run ./scripts/macosbundle <bin-path> <app-path> [version]
package main

import (
	"fmt"
	"os"

	"github.com/AlienFacepalm/giffer/internal/release/macosbundle"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: macosbundle <bin-path> <app-path> [version]\n")
		os.Exit(2)
	}
	version := macosbundle.ReadVersion("README.md")
	if len(os.Args) >= 4 {
		version = os.Args[3]
	}
	if err := macosbundle.Bundle(os.Args[1], os.Args[2], version); err != nil {
		fmt.Fprintf(os.Stderr, "macosbundle: %v\n", err)
		os.Exit(1)
	}
}
