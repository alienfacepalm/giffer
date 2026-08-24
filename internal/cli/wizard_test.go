package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestWizardBatchDefaults(t *testing.T) {
	root := t.TempDir()
	upload := filepath.Join(root, "upload")
	if err := os.MkdirAll(upload, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePNGZip(filepath.Join(upload, "photos.zip"), "a.png", 20, 10); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	// mode default, delay default, width default, loop default, confirm default
	in := strings.NewReader("\n\n\n\n\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	if err := runWizard(cmd, in, stdout); err != nil {
		t.Fatalf("wizard: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	out := filepath.Join("upload", "photos.gif")
	if !strings.Contains(stdout.String(), out) {
		t.Fatalf("stdout=%q want %s", stdout.String(), out)
	}
	if _, err := os.Stat(filepath.Join(root, out)); err != nil {
		t.Fatal(err)
	}
}

func TestWizardSingleConvert(t *testing.T) {
	root := t.TempDir()
	upload := filepath.Join(root, "upload")
	if err := os.MkdirAll(upload, 0o755); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(upload, "album.zip")
	if err := writePNGZip(zipPath, "a.png", 30, 15); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	script := strings.Join([]string{
		"2", // single
		"200",
		"0",
		"0",
		zipPath,
		"", // default output
		"y",
		"",
	}, "\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	if err := runWizard(cmd, strings.NewReader(script), stdout); err != nil {
		t.Fatalf("wizard: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	want := filepath.Join(upload, "album.gif")
	if !strings.Contains(stdout.String(), want) && !strings.Contains(stdout.String(), filepath.Join("upload", "album.gif")) {
		t.Fatalf("stdout=%q want output path", stdout.String())
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatal(err)
	}
}

func TestWizardCancel(t *testing.T) {
	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	if err := runWizard(cmd, strings.NewReader(""), stdout); err != nil {
		t.Fatalf("want cancel, got %v", err)
	}
	if !strings.Contains(stdout.String(), "Cancelled.") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestNoUserParams(t *testing.T) {
	cmd := newRootCmd()
	if !noUserParams(cmd) {
		t.Fatal("fresh command should have no user params")
	}
	_ = cmd.Flags().Set("delay-ms", "200")
	if noUserParams(cmd) {
		t.Fatal("Changed delay-ms should count as user params")
	}
}

func TestIsTerminalNonFile(t *testing.T) {
	if isTerminal(strings.NewReader("")) {
		t.Fatal("non-file reader must not look like a terminal")
	}
}
