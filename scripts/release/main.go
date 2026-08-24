// Command release builds giffer binaries for Windows, Linux, and macOS.
// Windows uses go-webview2 (no CGO). Linux and macOS use webview_go (CGO).
// Release binaries are native GUI apps: double-click opens the convert UI.
//
// Usage: go run ./scripts/release [os/arch ...]
// With no args, builds all supported targets. Examples:
//
//	go run ./scripts/release windows/amd64 linux/amd64
//	go run ./scripts/release linux/arm64
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/AlienFacepalm/giffer/internal/release/macosbundle"
)

type target struct {
	goos, goarch, dir, name string
}

func allTargets() []target {
	return []target{
		{"windows", "amd64", "windows-amd64", "giffer.exe"},
		{"windows", "386", "windows-386", "giffer.exe"},
		{"linux", "amd64", "linux-amd64", "giffer"},
		{"linux", "arm64", "linux-arm64", "giffer"},
		{"darwin", "amd64", "darwin-amd64", "giffer"},
		{"darwin", "arm64", "darwin-arm64", "giffer"},
	}
}

func selectTargets(args []string) []target {
	all := allTargets()
	if len(args) == 0 {
		return all
	}
	byKey := make(map[string]target, len(all))
	for _, t := range all {
		byKey[t.goos+"/"+t.goarch] = t
	}
	var out []target
	for _, a := range args {
		key := strings.TrimSpace(a)
		t, ok := byKey[key]
		if !ok {
			fail(fmt.Errorf("unknown target %q (want e.g. windows/amd64)", key))
		}
		out = append(out, t)
	}
	return out
}

func main() {
	version := macosbundle.ReadVersion("README.md")
	fmt.Printf("stamping version %s\n", version)

	targets := selectTargets(os.Args[1:])

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
		_ = os.Remove(out) // drop stale binary so a failed build cannot look fresh
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
			fail(fmt.Errorf("%s/%s: %w", t.goos, t.goarch, err))
		}

		if t.goos == "darwin" {
			appPath := filepath.Join(dir, "Giffer.app")
			if err := macosbundle.Bundle(out, appPath, version); err != nil {
				fail(fmt.Errorf("bundle %s: %w", appPath, err))
			}
			fmt.Printf("bundled %s\n", appPath)
		}
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "release: %v\n", err)
	os.Exit(1)
}
