package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	out, err := exec.Command("gofmt", "-l", ".").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gofmt: %v\n", err)
		os.Exit(1)
	}
	files := strings.TrimSpace(string(out))
	if files == "" {
		return
	}
	fmt.Fprintln(os.Stderr, "gofmt needed on:")
	fmt.Fprintln(os.Stderr, files)
	os.Exit(1)
}
