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

func TestToGIFFromZip(t *testing.T) {
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

	if err := ToGIF(Options{
		Inputs:   []string{zipPath},
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
	if g.Image[0].Bounds().Dx() != 80 {
		t.Fatalf("width=%d want 80 (resized)", g.Image[0].Bounds().Dx())
	}
}

func TestToGIFFromDirectory(t *testing.T) {
	dir := t.TempDir()
	photos := filepath.Join(dir, "photos")
	if err := os.MkdirAll(filepath.Join(photos, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(filepath.Join(photos, "b.png"), solid(30, 10, color.RGBA{255, 0, 0, 255})); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(filepath.Join(photos, "nested", "a.png"), solid(50, 10, color.RGBA{0, 255, 0, 255})); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(photos, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "photos.gif")
	if err := ToGIF(Options{
		Inputs:   []string{photos},
		Output:   outPath,
		DelayMS:  100,
		MaxWidth: 800,
		Loop:     0,
	}); err != nil {
		t.Fatal(err)
	}

	g := decodeGIF(t, outPath)
	if len(g.Image) != 2 {
		t.Fatalf("frames=%d want 2", len(g.Image))
	}
	if !isMostlyGreen(g.Image[0]) {
		t.Fatal("first frame should be a.png (green) after basename sort")
	}
}

func TestToGIFFromLooseFiles(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.png")
	b := filepath.Join(dir, "b.png")
	if err := writePNG(a, solid(40, 20, color.RGBA{0, 255, 0, 255})); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(b, solid(60, 20, color.RGBA{255, 0, 0, 255})); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "out.gif")
	if err := ToGIF(Options{
		Inputs:   []string{b, a},
		Output:   outPath,
		DelayMS:  200,
		MaxWidth: 800,
		Loop:     1,
	}); err != nil {
		t.Fatal(err)
	}

	g := decodeGIF(t, outPath)
	if len(g.Image) != 2 {
		t.Fatalf("frames=%d want 2", len(g.Image))
	}
	if !isMostlyGreen(g.Image[0]) {
		t.Fatal("first frame should be a.png (green) after basename sort")
	}
	if g.LoopCount != 1 {
		t.Fatalf("loop=%d want 1", g.LoopCount)
	}
}

func TestToGIFMixedSources(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "extra.zip")
	if err := writeTestZip(zipPath, map[string]imageEntry{
		"c.png": {img: solid(20, 10, color.White), format: "png"},
	}); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(dir, "a.png")
	if err := writePNG(filePath, solid(40, 10, color.White)); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "mix.gif")
	if err := ToGIF(Options{
		Inputs:   []string{zipPath, filePath},
		Output:   outPath,
		DelayMS:  500,
		MaxWidth: 800,
		Loop:     0,
	}); err != nil {
		t.Fatal(err)
	}

	g := decodeGIF(t, outPath)
	if len(g.Image) != 2 {
		t.Fatalf("frames=%d want 2", len(g.Image))
	}
}

func TestToGIFAcceptsJPEG(t *testing.T) {
	dir := t.TempDir()
	jpgPath := filepath.Join(dir, "frame.jpg")
	if err := writeJPEG(jpgPath, solid(64, 32, color.RGBA{10, 20, 30, 255})); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "out.gif")
	if err := ToGIF(Options{
		Inputs:   []string{jpgPath},
		Output:   outPath,
		DelayMS:  500,
		MaxWidth: 32,
		Loop:     0,
	}); err != nil {
		t.Fatal(err)
	}
	g := decodeGIF(t, outPath)
	if g.Image[0].Bounds().Dx() != 32 {
		t.Fatalf("width=%d want 32", g.Image[0].Bounds().Dx())
	}
}

func TestToGIFErrors(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing zip", func(t *testing.T) {
		err := ToGIF(Options{
			Inputs:   []string{filepath.Join(dir, "missing.zip")},
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
		err := ToGIF(Options{
			Inputs:   []string{path},
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
		err := ToGIF(Options{
			Inputs:   []string{path},
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
		err := ToGIF(Options{
			Inputs:   []string{path},
			Output:   filepath.Join(dir, "badframes.gif"),
			DelayMS:  500,
			MaxWidth: 800,
		})
		if err == nil || !strings.Contains(err.Error(), "no usable image frames") {
			t.Fatalf("want no usable frames error, got %v", err)
		}
	})

	t.Run("unsupported file type", func(t *testing.T) {
		path := filepath.Join(dir, "notes.txt")
		if err := os.WriteFile(path, []byte("hi"), 0o644); err != nil {
			t.Fatal(err)
		}
		err := ToGIF(Options{
			Inputs:   []string{path},
			Output:   filepath.Join(dir, "out.gif"),
			DelayMS:  500,
			MaxWidth: 800,
		})
		if err == nil || !strings.Contains(err.Error(), "unsupported input") {
			t.Fatalf("want unsupported input error, got %v", err)
		}
	})
}

type imageEntry struct {
	img    image.Image
	format string
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

func isMostlyGreen(img image.Image) bool {
	r, g, b, _ := img.At(img.Bounds().Min.X, img.Bounds().Min.Y).RGBA()
	return g > r && g > b
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

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func writeJPEG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
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
