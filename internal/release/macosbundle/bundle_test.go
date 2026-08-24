package macosbundle_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlienFacepalm/giffer/internal/release/macosbundle"
)

func TestBundleCreatesApp(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "giffer")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(dir, "Giffer.app")
	if err := macosbundle.Bundle(bin, app, "1.2.3"); err != nil {
		t.Fatal(err)
	}

	exe := filepath.Join(app, "Contents", "MacOS", "giffer")
	if _, err := os.Stat(exe); err != nil {
		t.Fatalf("missing bundled exe: %v", err)
	}

	plist, err := os.ReadFile(filepath.Join(app, "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(plist)
	if !strings.Contains(text, "1.2.3") {
		t.Fatalf("plist missing version: %s", text)
	}
	if !strings.Contains(text, "APPL") {
		t.Fatalf("plist missing APPL type: %s", text)
	}
}

func TestReadVersion(t *testing.T) {
	dir := t.TempDir()
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("**Release:** v9.8.7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := macosbundle.ReadVersion(readme); got != "9.8.7" {
		t.Fatalf("version=%q want 9.8.7", got)
	}
}
