package cli

import (
	"archive/zip"
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateOptionsDefaults(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "photos.zip")
	if err := os.WriteFile(zipPath, []byte("PK"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts, err := validateOptions([]string{zipPath}, "", 500, 800, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "photos.gif")
	if opts.Output != want {
		t.Fatalf("output=%q want %q", opts.Output, want)
	}
}

func TestValidateOptionsDirectoryDefault(t *testing.T) {
	dir := t.TempDir()
	photos := filepath.Join(dir, "album")
	if err := os.MkdirAll(photos, 0o755); err != nil {
		t.Fatal(err)
	}
	opts, err := validateOptions([]string{photos}, "", 500, 800, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "album.gif")
	if opts.Output != want {
		t.Fatalf("output=%q want %q", opts.Output, want)
	}
}

func TestValidateOptionsMultiDefault(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.png")
	b := filepath.Join(dir, "b.png")
	if err := os.WriteFile(a, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts, err := validateOptions([]string{a, b}, "", 500, 800, 0)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Output != "animation.gif" {
		t.Fatalf("output=%q want animation.gif", opts.Output)
	}
}

func TestValidateOptionsRejects(t *testing.T) {
	cases := []struct {
		name     string
		inputs   []string
		output   string
		delayMS  int
		maxWidth int
		loop     int
	}{
		{"empty inputs", nil, "", 500, 800, 0},
		{"unsupported ext", []string{"upload/a.txt"}, "", 500, 800, 0},
		{"delay", []string{"upload/a.zip"}, "", 0, 800, 0},
		{"width", []string{"upload/a.zip"}, "", 500, 0, 0},
		{"loop", []string{"upload/a.zip"}, "", 500, 800, -1},
		{"output ext", []string{"upload/a.zip"}, "out.png", 500, 800, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateOptions(tc.inputs, tc.output, tc.delayMS, tc.maxWidth, tc.loop)
			var inv *invalidParamsError
			if !errors.As(err, &inv) {
				t.Fatalf("want invalidParamsError, got %v", err)
			}
		})
	}
}

func TestRunExitCodes(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "photos.zip")
	outPath := filepath.Join(dir, "photos.gif")
	if err := writePNGZip(zipPath, "a.png", 40, 20); err != nil {
		t.Fatal(err)
	}

	t.Run("success zip flag", func(t *testing.T) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run([]string{
			"--input", zipPath,
			"--output", outPath,
			"--delay-ms", "500",
			"--max-width", "800",
			"--loop", "0",
		}, stdout, stderr)
		if code != exitOK {
			t.Fatalf("code=%d stderr=%s", code, stderr.String())
		}
		got := strings.TrimSpace(stdout.String())
		if got != outPath {
			t.Fatalf("stdout=%q want %q", got, outPath)
		}
	})

	t.Run("success positional images", func(t *testing.T) {
		imgDir := t.TempDir()
		a := filepath.Join(imgDir, "a.png")
		b := filepath.Join(imgDir, "b.png")
		if err := writePNGFile(a, 30, 10); err != nil {
			t.Fatal(err)
		}
		if err := writePNGFile(b, 30, 10); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(imgDir, "out.gif")
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run([]string{a, b, "--output", out}, stdout, stderr)
		if code != exitOK {
			t.Fatalf("code=%d stderr=%s", code, stderr.String())
		}
	})

	t.Run("success directory", func(t *testing.T) {
		imgDir := t.TempDir()
		if err := writePNGFile(filepath.Join(imgDir, "frame.png"), 20, 10); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(imgDir, "dir.gif")
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run([]string{"--input", imgDir, "--output", out}, stdout, stderr)
		if code != exitOK {
			t.Fatalf("code=%d stderr=%s", code, stderr.String())
		}
	})

	t.Run("overwrite warning", func(t *testing.T) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run([]string{"--input", zipPath, "--output", outPath}, stdout, stderr)
		if code != exitOK {
			t.Fatalf("code=%d stderr=%s", code, stderr.String())
		}
		if !strings.Contains(stderr.String(), "warning: overwriting") {
			t.Fatalf("expected overwrite warning, got %q", stderr.String())
		}
	})

	t.Run("invalid params", func(t *testing.T) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run([]string{"--input", "nope.txt"}, stdout, stderr)
		if code != exitInvalidParams {
			t.Fatalf("code=%d want %d stderr=%s", code, exitInvalidParams, stderr.String())
		}
	})

	t.Run("missing input", func(t *testing.T) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run(nil, stdout, stderr)
		if code != exitInvalidParams {
			t.Fatalf("code=%d want %d stderr=%s", code, exitInvalidParams, stderr.String())
		}
	})

	t.Run("runtime missing zip", func(t *testing.T) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run([]string{"--input", filepath.Join(dir, "missing.zip")}, stdout, stderr)
		if code != exitRuntime {
			t.Fatalf("code=%d want %d stderr=%s", code, exitRuntime, stderr.String())
		}
		if !strings.Contains(stderr.String(), "unreadable zip") {
			t.Fatalf("stderr=%q", stderr.String())
		}
	})
}

func writePNGZip(path, name string, w, h int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	wr, err := zw.Create(name)
	if err != nil {
		return err
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.White)
		}
	}
	if err := png.Encode(wr, img); err != nil {
		return err
	}
	return zw.Close()
}

func writePNGFile(path string, w, h int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.White)
		}
	}
	return png.Encode(f, img)
}
