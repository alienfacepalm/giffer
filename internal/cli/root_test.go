package cli

import (
	"archive/zip"
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
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

func TestValidateOptionsDirectory(t *testing.T) {
	dir := t.TempDir()
	album := filepath.Join(dir, "album")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	opts, err := validateOptions(album, "", 500, 800, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "album.gif")
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
		{"width", "upload/a.zip", "", 500, -1, 0},
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
		}, strings.NewReader(""), stdout, stderr)
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
		}, strings.NewReader(""), stdout, stderr)
		if code != exitOK {
			t.Fatalf("code=%d stderr=%s", code, stderr.String())
		}
		if !strings.Contains(stderr.String(), "warning: overwriting") {
			t.Fatalf("expected overwrite warning, got %q", stderr.String())
		}
	})

	t.Run("directory input", func(t *testing.T) {
		album := filepath.Join(dir, "album")
		if err := os.MkdirAll(album, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := writePNGFile(filepath.Join(album, "a.png"), 30, 15); err != nil {
			t.Fatal(err)
		}
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run([]string{"--input", album}, strings.NewReader(""), stdout, stderr)
		if code != exitOK {
			t.Fatalf("code=%d stderr=%s", code, stderr.String())
		}
		want := filepath.Join(dir, "album.gif")
		if strings.TrimSpace(stdout.String()) != want {
			t.Fatalf("stdout=%q want %q", stdout.String(), want)
		}
	})

	t.Run("invalid params", func(t *testing.T) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run([]string{"--input", "nope.txt"}, strings.NewReader(""), stdout, stderr)
		if code != exitInvalidParams {
			t.Fatalf("code=%d want %d stderr=%s", code, exitInvalidParams, stderr.String())
		}
	})

	t.Run("runtime unreadable zip", func(t *testing.T) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Run([]string{
			"--input", filepath.Join(dir, "missing.zip"),
		}, strings.NewReader(""), stdout, stderr)
		if code != exitRuntime {
			t.Fatalf("code=%d want %d stderr=%s", code, exitRuntime, stderr.String())
		}
		if !strings.Contains(stderr.String(), "unreadable archive") {
			t.Fatalf("stderr=%q", stderr.String())
		}
	})
}

func TestRunBatch(t *testing.T) {
	root := t.TempDir()
	upload := filepath.Join(root, "upload")
	if err := os.MkdirAll(upload, 0o755); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(upload, "photos.zip")
	if err := writePNGZip(zipPath, "a.png", 40, 20); err != nil {
		t.Fatal(err)
	}

	album := filepath.Join(upload, "vacation")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePNGFile(filepath.Join(album, "b.png"), 40, 20); err != nil {
		t.Fatal(err)
	}

	empty := filepath.Join(upload, "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}

	existing := filepath.Join(upload, "done.gif")
	if err := os.WriteFile(existing, []byte("gif"), 0o644); err != nil {
		t.Fatal(err)
	}
	doneZip := filepath.Join(upload, "done.zip")
	if err := writePNGZip(doneZip, "c.png", 20, 10); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Run(nil, strings.NewReader(""), stdout, stderr)
	if code != exitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "skip "+filepath.Join("upload", "done.gif")) {
		t.Fatalf("expected skip for done.gif, got %q", out)
	}
	photosGIF := filepath.Join("upload", "photos.gif")
	vacationGIF := filepath.Join("upload", "vacation.gif")
	if !strings.Contains(out, photosGIF) {
		t.Fatalf("expected created %s in %q", photosGIF, out)
	}
	if !strings.Contains(out, vacationGIF) {
		t.Fatalf("expected created %s in %q", vacationGIF, out)
	}
	if _, err := os.Stat(filepath.Join(root, photosGIF)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, vacationGIF)); err != nil {
		t.Fatal(err)
	}

	// Second run: everything skipped.
	stdout2, stderr2 := &bytes.Buffer{}, &bytes.Buffer{}
	code = Run(nil, strings.NewReader(""), stdout2, stderr2)
	if code != exitOK {
		t.Fatalf("second run code=%d stderr=%s", code, stderr2.String())
	}
	if !strings.Contains(stdout2.String(), "skip") {
		t.Fatalf("expected skips on second run, got %q", stdout2.String())
	}
}

func TestRunBatchEmptyUpload(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "upload"), 0o755); err != nil {
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

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Run(nil, strings.NewReader(""), stdout, stderr)
	if code != exitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}

func TestRunBatchMissingUpload(t *testing.T) {
	root := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Run(nil, strings.NewReader(""), stdout, stderr)
	if code != exitRuntime {
		t.Fatalf("code=%d want %d stderr=%s", code, exitRuntime, stderr.String())
	}
	if !strings.Contains(stderr.String(), "upload") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestRunBatchPartialFailure(t *testing.T) {
	root := t.TempDir()
	upload := filepath.Join(root, "upload")
	if err := os.MkdirAll(upload, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePNGZip(filepath.Join(upload, "good.zip"), "a.png", 20, 10); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(upload, "bad.zip"), []byte("not-a-zip"), 0o644); err != nil {
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

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Run(nil, strings.NewReader(""), stdout, stderr)
	if code != exitRuntime {
		t.Fatalf("code=%d want %d stderr=%s", code, exitRuntime, stderr.String())
	}
	if !strings.Contains(stderr.String(), "bad.zip") {
		t.Fatalf("expected per-job error for bad.zip, stderr=%q", stderr.String())
	}
	if !strings.Contains(stdout.String(), filepath.Join("upload", "good.gif")) {
		t.Fatalf("expected good.gif on stdout, got %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, "upload", "good.gif")); err != nil {
		t.Fatal(err)
	}
}

func TestRunBatchTunables(t *testing.T) {
	root := t.TempDir()
	upload := filepath.Join(root, "upload")
	if err := os.MkdirAll(upload, 0o755); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(upload, "wide.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, name := range []string{"a.png", "b.png"} {
		wr, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		img := image.NewRGBA(image.Rect(0, 0, 100, 40))
		for y := 0; y < 40; y++ {
			for x := 0; x < 100; x++ {
				img.Set(x, y, color.White)
			}
		}
		if err := png.Encode(wr, img); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
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

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Run([]string{"--delay-ms", "200", "--max-width", "50", "--loop", "3"}, strings.NewReader(""), stdout, stderr)
	if code != exitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	outPath := filepath.Join(root, "upload", "wide.gif")
	outFile, err := os.Open(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer outFile.Close()
	g, err := gif.DecodeAll(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Image) != 2 {
		t.Fatalf("frames=%d want 2", len(g.Image))
	}
	if g.LoopCount != 3 {
		t.Fatalf("loop=%d want 3", g.LoopCount)
	}
	if g.Delay[0] != 20 {
		t.Fatalf("delay=%d want 20", g.Delay[0])
	}
	if g.Image[0].Bounds().Dx() != 50 {
		t.Fatalf("width=%d want 50", g.Image[0].Bounds().Dx())
	}
}

func TestRunBatchCollision(t *testing.T) {
	root := t.TempDir()
	upload := filepath.Join(root, "upload")
	if err := os.MkdirAll(upload, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePNGZip(filepath.Join(upload, "photos.zip"), "a.png", 20, 10); err != nil {
		t.Fatal(err)
	}
	album := filepath.Join(upload, "photos")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePNGFile(filepath.Join(album, "b.png"), 20, 10); err != nil {
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

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Run(nil, strings.NewReader(""), stdout, stderr)
	if code != exitRuntime {
		t.Fatalf("code=%d want %d stderr=%s", code, exitRuntime, stderr.String())
	}
	if !strings.Contains(stderr.String(), "output collision") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestRunUIHelp(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Run([]string{"ui", "--help"}, strings.NewReader(""), stdout, stderr)
	if code != exitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "native window") {
		t.Fatalf("help=%q", stdout.String())
	}
}

func TestDiscoverUploadJobs(t *testing.T) {
	dir := t.TempDir()
	if err := writePNGZip(filepath.Join(dir, "a.zip"), "x.png", 10, 10); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePNGFile(filepath.Join(sub, "y.png"), 10, 10); err != nil {
		t.Fatal(err)
	}
	jobs, err := discoverUploadJobs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("jobs=%d want 2", len(jobs))
	}
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
