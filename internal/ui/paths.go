package ui

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultUploadDir returns upload/ beside the release binary. During go test
// or go run the path falls back to upload/ in the working directory.
func DefaultUploadDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "upload"
	}
	dir := filepath.Dir(exe)
	if isGoBuildBinary(dir) {
		return "upload"
	}
	return filepath.Join(dir, "upload")
}

func isGoBuildBinary(dir string) bool {
	// Temp dirs from `go run` / `go test` on Windows, Linux, and macOS.
	return strings.Contains(filepath.ToSlash(dir), "/go-build")
}
