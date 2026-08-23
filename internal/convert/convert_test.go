package convert

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShouldSkipPath(t *testing.T) {
	cases := []struct {
		full string
		base string
		skip bool
	}{
		{"photos/a.jpg", "a.jpg", false},
		{"__MACOSX/._a.jpg", "._a.jpg", true},
		{"photos/.DS_Store", ".DS_Store", true},
		{"photos/.hidden.png", ".hidden.png", true},
		{"nested/dir/b.PNG", "b.PNG", false},
	}
	for _, tc := range cases {
		if got := shouldSkipPath(tc.full, tc.base); got != tc.skip {
			t.Fatalf("%s: skip=%v want %v", tc.full, got, tc.skip)
		}
	}
}

func TestIsSupportedImage(t *testing.T) {
	ok := []string{"a.jpg", "b.JPEG", "c.png", "d.webp", "e.gif"}
	for _, name := range ok {
		if !isSupportedImage(name) {
			t.Fatalf("expected supported: %s", name)
		}
	}
	bad := []string{"notes.txt", "a.bmp", "a.tiff", "a"}
	for _, name := range bad {
		if isSupportedImage(name) {
			t.Fatalf("expected unsupported: %s", name)
		}
	}
}

func TestZipToGIFSuccess(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "photos.zip")
	outPath := filepath.Join(dir, "photos.gif")

	if err := writeTestZip(zipPath, map[string]imageEntry{
		"__MACOSX/._noise.jpg": {img: solid(10, 10, color.RGBA{1, 0, 0, 255}), format: "png"},
		"photos/z.png":         {img: solid(100, 50, color.RGBA{255, 0, 0, 255}), format: "png"},
		"photos/a.png":         {img: solid(120, 60, color.RGBA{0, 255, 0, 255}), format: "png"},
		"readme.txt":           {raw: []byte("not an image")},
	}); err != nil {
		t.Fatal(err)
	}

	if err := ZipToGIF(Options{
		Input:    zipPath,
		Output:   outPath,
		DelayMS:  500,
		MaxWidth: 80,
		Loop:     0,
	}); err != nil {
		t.Fatal(err)
	}

	g := decodeGIF(t, outPath)
	if len(g.Image) != 2 {
		t.Fatalf("frames=%d want 2", len(g.Image))
	}
	if g.Delay[0] != 50 {
		t.Fatalf("delay=%d want 50 centiseconds", g.Delay[0])
	}
	if g.LoopCount != 0 {
		t.Fatalf("loop=%d want 0", g.LoopCount)
	}
	if g.Image[0].Bounds().Dx() != 80 {
		t.Fatalf("width=%d want 80 (resized)", g.Image[0].Bounds().Dx())
	}
}

func TestZipToGIFSortsCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "photos.zip")
	outPath := filepath.Join(dir, "out.gif")

	// Case-insensitive order: a.png then B.png.
	// Distinct widths so we can assert order without relying on pixel colors.
	if err := writeTestZip(zipPath, map[string]imageEntry{
		"nested/B.png": {img: solid(30, 10, color.White), format: "png"},
		"nested/a.png": {img: solid(50, 10, color.White), format: "png"},
	}); err != nil {
		t.Fatal(err)
	}

	if err := ZipToGIF(Options{
		Input:    zipPath,
		Output:   outPath,
		DelayMS:  100,
		MaxWidth: 800,
		Loop:     2,
	}); err != nil {
		t.Fatal(err)
	}

	g := decodeGIF(t, outPath)
	if len(g.Image) != 2 {
		t.Fatalf("frames=%d want 2", len(g.Image))
	}
	if g.Image[0].Bounds().Dx() != 50 {
		t.Fatalf("first frame width=%d want 50 (a.png first)", g.Image[0].Bounds().Dx())
	}
	if g.Image[1].Bounds().Dx() != 30 {
		t.Fatalf("second frame width=%d want 30 (B.png second)", g.Image[1].Bounds().Dx())
	}
	if g.LoopCount != 2 {
		t.Fatalf("loop=%d want 2", g.LoopCount)
	}
}

func TestZipToGIFLeavesNarrowImagesUnscaled(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "photos.zip")
	outPath := filepath.Join(dir, "out.gif")

	if err := writeTestZip(zipPath, map[string]imageEntry{
		"small.png": {img: solid(40, 20, color.White), format: "png"},
	}); err != nil {
		t.Fatal(err)
	}

	if err := ZipToGIF(Options{
		Input:    zipPath,
		Output:   outPath,
		DelayMS:  200,
		MaxWidth: 800,
		Loop:     0,
	}); err != nil {
		t.Fatal(err)
	}

	g := decodeGIF(t, outPath)
	if g.Image[0].Bounds().Dx() != 40 {
		t.Fatalf("width=%d want 40 (unscaled)", g.Image[0].Bounds().Dx())
	}
	if g.Delay[0] != 20 {
		t.Fatalf("delay=%d want 20", g.Delay[0])
	}
}

func TestZipToGIFAcceptsJPEG(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "photos.zip")
	outPath := filepath.Join(dir, "out.gif")

	if err := writeTestZip(zipPath, map[string]imageEntry{
		"frame.jpg": {img: solid(64, 32, color.RGBA{10, 20, 30, 255}), format: "jpeg"},
	}); err != nil {
		t.Fatal(err)
	}

	if err := ZipToGIF(Options{
		Input:    zipPath,
		Output:   outPath,
		DelayMS:  500,
		MaxWidth: 32,
		Loop:     0,
	}); err != nil {
		t.Fatal(err)
	}

	g := decodeGIF(t, outPath)
	if len(g.Image) != 1 {
		t.Fatalf("frames=%d want 1", len(g.Image))
	}
	if g.Image[0].Bounds().Dx() != 32 {
		t.Fatalf("width=%d want 32", g.Image[0].Bounds().Dx())
	}
}

func TestDirToGIFSuccess(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "photos")
	outPath := filepath.Join(dir, "photos.gif")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePNGFile(filepath.Join(src, "nested", "z.png"), 100, 50); err != nil {
		t.Fatal(err)
	}
	if err := writePNGFile(filepath.Join(src, "a.png"), 120, 60); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "readme.txt"), []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "__MACOSX"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePNGFile(filepath.Join(src, "__MACOSX", "skip.png"), 8, 8); err != nil {
		t.Fatal(err)
	}

	if err := DirToGIF(Options{
		Input:    src,
		Output:   outPath,
		DelayMS:  500,
		MaxWidth: 80,
		Loop:     0,
	}); err != nil {
		t.Fatal(err)
	}

	g := decodeGIF(t, outPath)
	if len(g.Image) != 2 {
		t.Fatalf("frames=%d want 2", len(g.Image))
	}
	if g.Image[0].Bounds().Dx() != 80 {
		t.Fatalf("width=%d want 80 (resized)", g.Image[0].Bounds().Dx())
	}
}

func TestDirToGIFEmpty(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "empty")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "readme.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := DirToGIF(Options{
		Input:    src,
		Output:   filepath.Join(dir, "empty.gif"),
		DelayMS:  500,
		MaxWidth: 800,
	})
	if err == nil || !strings.Contains(err.Error(), "no supported images") {
		t.Fatalf("want no supported images error, got %v", err)
	}
}

func TestHasImages(t *testing.T) {
	dir := t.TempDir()
	with := filepath.Join(dir, "with")
	without := filepath.Join(dir, "without")
	if err := os.MkdirAll(with, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(without, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePNGFile(filepath.Join(with, "a.png"), 10, 10); err != nil {
		t.Fatal(err)
	}
	if !HasImages(with) {
		t.Fatal("expected HasImages true")
	}
	if HasImages(without) {
		t.Fatal("expected HasImages false")
	}
}

func TestConvertDispatches(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "photos.zip")
	dirPath := filepath.Join(dir, "album")
	if err := writeTestZip(zipPath, map[string]imageEntry{
		"a.png": {img: solid(20, 10, color.White), format: "png"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePNGFile(filepath.Join(dirPath, "b.png"), 20, 10); err != nil {
		t.Fatal(err)
	}

	zipOut := filepath.Join(dir, "fromzip.gif")
	if err := Convert(Options{Input: zipPath, Output: zipOut, DelayMS: 100, MaxWidth: 800, Loop: 0}); err != nil {
		t.Fatal(err)
	}
	dirOut := filepath.Join(dir, "fromdir.gif")
	if err := Convert(Options{Input: dirPath, Output: dirOut, DelayMS: 100, MaxWidth: 800, Loop: 0}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(zipOut); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dirOut); err != nil {
		t.Fatal(err)
	}
}

func TestZipToGIFErrors(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing zip", func(t *testing.T) {
		err := ZipToGIF(Options{
			Input:    filepath.Join(dir, "missing.zip"),
			Output:   filepath.Join(dir, "out.gif"),
			DelayMS:  500,
			MaxWidth: 800,
		})
		if err == nil || !strings.Contains(err.Error(), "unreadable zip") {
			t.Fatalf("want unreadable zip error, got %v", err)
		}
	})

	t.Run("corrupt zip", func(t *testing.T) {
		path := filepath.Join(dir, "corrupt.zip")
		if err := os.WriteFile(path, []byte("not a zip"), 0o644); err != nil {
			t.Fatal(err)
		}
		err := ZipToGIF(Options{
			Input:    path,
			Output:   filepath.Join(dir, "corrupt.gif"),
			DelayMS:  500,
			MaxWidth: 800,
		})
		if err == nil || !strings.Contains(err.Error(), "unreadable zip") {
			t.Fatalf("want unreadable zip error, got %v", err)
		}
	})

	t.Run("empty of images", func(t *testing.T) {
		path := filepath.Join(dir, "empty.zip")
		if err := writeTestZip(path, map[string]imageEntry{
			"readme.txt":          {raw: []byte("hi")},
			"__MACOSX/._skip.png": {img: solid(8, 8, color.White), format: "png"},
			"photos/.DS_Store":    {raw: []byte("store")},
		}); err != nil {
			t.Fatal(err)
		}
		err := ZipToGIF(Options{
			Input:    path,
			Output:   filepath.Join(dir, "empty.gif"),
			DelayMS:  500,
			MaxWidth: 800,
		})
		if err == nil || !strings.Contains(err.Error(), "no supported images") {
			t.Fatalf("want no supported images error, got %v", err)
		}
	})

	t.Run("no usable frames", func(t *testing.T) {
		path := filepath.Join(dir, "badframes.zip")
		if err := writeTestZip(path, map[string]imageEntry{
			"broken.jpg": {raw: []byte("not-a-real-jpeg")},
			"broken.png": {raw: []byte("not-a-real-png")},
		}); err != nil {
			t.Fatal(err)
		}
		err := ZipToGIF(Options{
			Input:    path,
			Output:   filepath.Join(dir, "badframes.gif"),
			DelayMS:  500,
			MaxWidth: 800,
		})
		if err == nil || !strings.Contains(err.Error(), "no usable image frames") {
			t.Fatalf("want no usable frames error, got %v", err)
		}
	})
}

type imageEntry struct {
	img    image.Image
	format string // png or jpeg
	raw    []byte
}

func solid(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func decodeGIF(t *testing.T, path string) *gif.GIF {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func writePNGFile(path string, w, h int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, solid(w, h, color.White))
}

func writeTestZip(path string, files map[string]imageEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for name, entry := range files {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if entry.raw != nil {
			if _, err := w.Write(entry.raw); err != nil {
				return err
			}
			continue
		}
		var buf bytes.Buffer
		switch entry.format {
		case "jpeg", "jpg":
			if err := jpeg.Encode(&buf, entry.img, &jpeg.Options{Quality: 90}); err != nil {
				return err
			}
		default:
			if err := png.Encode(&buf, entry.img); err != nil {
				return err
			}
		}
		if _, err := w.Write(buf.Bytes()); err != nil {
			return err
		}
	}
	return zw.Close()
}
