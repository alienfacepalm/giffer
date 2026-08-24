// Command release builds giffer binaries for Windows, Linux, and macOS.
// Windows uses go-webview2 (no CGO). Linux and macOS use webview_go (CGO).
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	targets := []struct {
		goos, goarch, dir, name string
	}{
		{"windows", "amd64", "windows-amd64", "giffer.exe"},
		{"windows", "386", "windows-386", "giffer.exe"},
		{"linux", "amd64", "linux-amd64", "giffer"},
		{"linux", "arm64", "linux-arm64", "giffer"},
		{"darwin", "amd64", "darwin-amd64", "giffer"},
		{"darwin", "arm64", "darwin-arm64", "giffer"},
	}

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

	hostGOOS := runtime.GOOS

	for _, t := range targets {
		dir := filepath.Join("release", t.dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fail(err)
		}
		out := filepath.Join(dir, t.name)
		fmt.Printf("building %s/%s → %s\n", t.goos, t.goarch, out)

		ldflags := "-s -w"
		if t.goos == "windows" {
			ldflags += " -H=windowsgui"
		}

		cmd := exec.Command("go", "build", "-tags", "desktop", "-trimpath", "-ldflags="+ldflags, "-o", out, "./cmd/giffer")

		cgo := "1"
		if t.goos == "windows" {
			cgo = "0"
		}

		env := append(os.Environ(),
			"CGO_ENABLED="+cgo,
			"GOOS="+t.goos,
			"GOARCH="+t.goarch,
		)

		if hostGOOS != t.goos {
			switch {
			case t.goos == "windows" && t.goarch == "amd64":
				env = append(env, "CC=x86_64-w64-mingw32-gcc")
			case t.goos == "windows" && t.goarch == "386":
				env = append(env, "CC=i686-w64-mingw32-gcc")
			}
		}

		cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s/%s: %v (skipped)\n", t.goos, t.goarch, err)
			continue
		}
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "release: %v\n", err)
	os.Exit(1)
}
