// Command build-gui builds a release-style native GUI binary into bin/.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	if err := os.MkdirAll("bin", 0o755); err != nil {
		fail(err)
	}

	name := "giffer"
	if runtime.GOOS == "windows" {
		name = "giffer.exe"
	}
	out := filepath.Join("bin", name)

	ldflags := "-s -w"
	if runtime.GOOS == "windows" {
		ldflags += " -H=windowsgui"
	}

	cmd := exec.Command("go", "build", "-tags", "desktop", "-trimpath", "-ldflags="+ldflags, "-o", out, "./cmd/giffer")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if runtime.GOOS != "windows" {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=1")
	} else {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=0")
	}

	fmt.Printf("building GUI binary → %s\n", out)
	if err := cmd.Run(); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "build-gui: %v\n", err)
	os.Exit(1)
}
