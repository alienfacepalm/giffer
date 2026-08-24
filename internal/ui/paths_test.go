package ui

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultUploadDirGoBuildUsesCWDRelative(t *testing.T) {
	if !isGoBuildBinary(filepath.Join("C:", "Temp", "go-build123", "b001", "exe")) {
		t.Fatal("expected go-build path detection")
	}
	if isGoBuildBinary(filepath.Join("C:", "Program Files", "giffer")) {
		t.Fatal("release install path should not look like go-build")
	}
}

func TestDefaultUploadDirReturnsNonEmpty(t *testing.T) {
	dir := DefaultUploadDir()
	if strings.TrimSpace(dir) == "" {
		t.Fatal("DefaultUploadDir returned empty string")
	}
}
