package convert

import (
	"archive/zip"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// Options controls a conversion run from a zip or photo directory.
type Options struct {
	Input    string
	Output   string
	DelayMS  int
	MaxWidth int
	Loop     int
}

// Convert reads images from a zip archive or directory and writes an animated GIF.
func Convert(opts Options) error {
	info, err := os.Stat(opts.Input)
	if err != nil {
		if strings.EqualFold(filepath.Ext(opts.Input), ".zip") {
			return fmt.Errorf("unreadable zip: %w", err)
		}
		return fmt.Errorf("unreadable input: %w", err)
	}
	if info.IsDir() {
		return DirToGIF(opts)
	}
	return ZipToGIF(opts)
}

// ZipToGIF reads images from a zip archive and writes an animated GIF.
func ZipToGIF(opts Options) error {
	zr, err := zip.OpenReader(opts.Input)
	if err != nil {
		return fmt.Errorf("unreadable zip: %w", err)
	}
	defer zr.Close()

	entries := collectZipImageEntries(zr.File)
	if len(entries) == 0 {
		return fmt.Errorf("no supported images found in zip")
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].base) < strings.ToLower(entries[j].base)
	})

	frames := make([]image.Image, 0, len(entries))
	for _, e := range entries {
		img, err := decodeZipImage(e.file)
		if err != nil {
			continue // skip unreadable frames; require at least one usable frame below
		}
		frames = append(frames, img)
	}
	return writeGIF(opts, frames)
}

// DirToGIF reads images from a directory (recursively) and writes an animated GIF.
func DirToGIF(opts Options) error {
	entries, err := collectDirImageEntries(opts.Input)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("no supported images found in directory")
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].base) < strings.ToLower(entries[j].base)
	})

	frames := make([]image.Image, 0, len(entries))
	for _, e := range entries {
		img, err := decodeFileImage(e.path)
		if err != nil {
			continue
		}
		frames = append(frames, img)
	}
	return writeGIF(opts, frames)
}

// HasImages reports whether dir contains at least one supported image file.
func HasImages(dir string) bool {
	entries, err := collectDirImageEntries(dir)
	return err == nil && len(entries) > 0
}

func writeGIF(opts Options, frames []image.Image) error {
	if len(frames) == 0 {
		return fmt.Errorf("no usable image frames")
	}

	delayCS := (opts.DelayMS + 5) / 10 // GIF delay is in 100ths of a second
	if delayCS < 1 {
		delayCS = 1
	}

	out := &gif.GIF{LoopCount: opts.Loop}
	for _, img := range frames {
		resized := resizeMaxWidth(img, opts.MaxWidth)
		paletted := quantize(resized)
		out.Image = append(out.Image, paletted)
		out.Delay = append(out.Delay, delayCS)
	}

	if err := os.MkdirAll(filepath.Dir(opts.Output), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	f, err := os.Create(opts.Output)
	if err != nil {
		return fmt.Errorf("write gif: %w", err)
	}
	defer f.Close()

	if err := gif.EncodeAll(f, out); err != nil {
		return fmt.Errorf("encode gif: %w", err)
	}
	return nil
}

type zipImage struct {
	file *zip.File
	base string
}

type dirImage struct {
	path string
	base string
}

func collectZipImageEntries(files []*zip.File) []zipImage {
	var out []zipImage
	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}
		name := f.Name
		base := path.Base(name)
		if shouldSkipPath(name, base) {
			continue
		}
		if !isSupportedImage(base) {
			continue
		}
		out = append(out, zipImage{file: f, base: base})
	}
	return out
}

func collectDirImageEntries(root string) ([]dirImage, error) {
	var out []dirImage
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		base := filepath.Base(p)

		if d.IsDir() {
			if p == root {
				return nil
			}
			if shouldSkipPath(relSlash, base) {
				return filepath.SkipDir
			}
			return nil
		}

		if shouldSkipPath(relSlash, base) {
			return nil
		}
		if !isSupportedImage(base) {
			return nil
		}
		out = append(out, dirImage{path: p, base: base})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	return out, nil
}

func shouldSkipPath(full, base string) bool {
	normalized := path.Clean("/" + strings.ReplaceAll(full, "\\", "/"))
	if strings.HasPrefix(base, ".") {
		return true
	}
	if strings.EqualFold(base, ".DS_Store") {
		return true
	}
	parts := strings.Split(strings.TrimPrefix(normalized, "/"), "/")
	for _, p := range parts {
		if p == "__MACOSX" {
			return true
		}
	}
	return false
}

func isSupportedImage(base string) bool {
	switch strings.ToLower(path.Ext(base)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return true
	default:
		return false
	}
}

func decodeZipImage(f *zip.File) (image.Image, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return decodeImage(rc)
}

func decodeFileImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return decodeImage(f)
}

func decodeImage(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func resizeMaxWidth(src image.Image, maxWidth int) image.Image {
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= maxWidth {
		return src
	}
	newW := maxWidth
	newH := h * newW / w
	if newH < 1 {
		newH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst
}

func quantize(src image.Image) *image.Paletted {
	b := src.Bounds()
	dst := image.NewPaletted(b, palette.Plan9)
	draw.FloydSteinberg.Draw(dst, b, src, b.Min)
	return dst
}
