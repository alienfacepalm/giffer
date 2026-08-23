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
	opts, err := validateOptions("upload/photos.zip", "", 500, 800, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("upload", "photos.gif")
	if opts.Output != want {
		t.Fatalf("output=%q want %q", opts.Output, want)
	}
}

func TestValidateOptionsRejects(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		output   string
		delayMS  int
		maxWidth int
		loop     int
	}{
		{"empty input", "", "", 500, 800, 0},
		{"not zip", "upload/a.txt", "", 500, 800, 0},
		{"delay", "upload/a.zip", "", 0, 800, 0},
		{"width", "upload/a.zip", "", 500, 0, 0},
		{"loop", "upload/a.zip", "", 500, 800, -1},
		{"output ext", "upload/a.zip", "out.png", 500, 800, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateOptions(tc.input, tc.output, tc.delayMS, tc.maxWidth, tc.loop)
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

	t.Run("success", func(t *testing.T) {
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
		if _, err := os.Stat(outPath); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("overwrite warning", func(t *testing.T) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run([]string{
			"--input", zipPath,
			"--output", outPath,
		}, stdout, stderr)
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

	t.Run("missing input flag", func(t *testing.T) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run(nil, stdout, stderr)
		if code != exitInvalidParams {
			t.Fatalf("code=%d want %d stderr=%s", code, exitInvalidParams, stderr.String())
		}
	})

	t.Run("runtime unreadable zip", func(t *testing.T) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run([]string{
			"--input", filepath.Join(dir, "missing.zip"),
		}, stdout, stderr)
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
