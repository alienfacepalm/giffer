// Command release builds static giffer binaries for Windows, Linux, and macOS.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	targets := []struct {
		goos, goarch, dir, name string
	}{
		{"windows", "amd64", "windows-amd64", "giffer.exe"},
		{"linux", "amd64", "linux-amd64", "giffer"},
		{"linux", "arm64", "linux-arm64", "giffer"},
		{"darwin", "amd64", "darwin-amd64", "giffer"},
		{"darwin", "arm64", "darwin-arm64", "giffer"},
	}

	// Remove legacy flat binaries from older layouts.
	for _, legacy := range []string{
		"giffer-windows-amd64.exe",
		"giffer-linux-amd64",
		"giffer-linux-arm64",
		"giffer-darwin-amd64",
		"giffer-darwin-arm64",
	} {
		_ = os.Remove(filepath.Join("release", legacy))
	}

	if err := os.MkdirAll("release", 0o755); err != nil {
		fail(err)
	}

	for _, t := range targets {
		dir := filepath.Join("release", t.dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fail(err)
		}
		out := filepath.Join(dir, t.name)
		fmt.Printf("building %s/%s → %s\n", t.goos, t.goarch, out)
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", out, "./cmd/giffer")
		cmd.Env = append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS="+t.goos,
			"GOARCH="+t.goarch,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fail(fmt.Errorf("%s/%s: %w", t.goos, t.goarch, err))
		}
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "release: %v\n", err)
	os.Exit(1)
}
