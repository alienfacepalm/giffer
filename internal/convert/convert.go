package convert

import (
	"archive/zip"
	"bytes"
	"context"
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
	Input      string
	Output     string
	DelayMS    int
	MaxWidth   int // 0 = use first sorted frame's native width
	Loop       int
	Ctx        context.Context // optional; cancel stops conversion when possible
	OnProgress func(Progress)  // optional; called as work advances
}

// Progress is a conversion status update for UI / logging.
type Progress struct {
	Stage   string // "reading", "encoding", "writing"
	Done    int
	Total   int
	Percent int // overall 0–100 across stages
}

func report(opts Options, stage string, done, total int) {
	if opts.OnProgress == nil || total < 1 {
		return
	}
	if done < 0 {
		done = 0
	}
	if done > total {
		done = total
	}
	opts.OnProgress(Progress{
		Stage:   stage,
		Done:    done,
		Total:   total,
		Percent: overallPercent(stage, done, total),
	})
}

func overallPercent(stage string, done, total int) int {
	frac := float64(done) / float64(total)
	switch stage {
	case "reading":
		return int(frac * 45)
	case "encoding":
		return 45 + int(frac*50)
	case "writing":
		return 95 + int(frac*5)
	default:
		return int(frac * 100)
	}
}

func validateOptions(opts Options) error {
	if opts.DelayMS <= 0 {
		return fmt.Errorf("delay must be > 0 (got %d)", opts.DelayMS)
	}
	if opts.MaxWidth < 0 {
		return fmt.Errorf("max-width must be >= 0 (got %d)", opts.MaxWidth)
	}
	if opts.Loop < 0 {
		return fmt.Errorf("loop must be >= 0 (got %d)", opts.Loop)
	}
	return nil
}

func checkCtx(opts Options) error {
	if opts.Ctx == nil {
		return nil
	}
	select {
	case <-opts.Ctx.Done():
		return opts.Ctx.Err()
	default:
		return nil
	}
}

// Convert reads images from a supported archive or directory and writes an animated GIF.
func Convert(opts Options) error {
	if err := validateOptions(opts); err != nil {
		return err
	}
	info, err := os.Stat(opts.Input)
	if err != nil {
		if IsArchive(opts.Input) {
			return fmt.Errorf("unreadable archive: %w", err)
		}
		return fmt.Errorf("unreadable input: %w", err)
	}
	if info.IsDir() {
		return DirToGIF(opts)
	}
	if IsArchive(opts.Input) {
		return ArchiveToGIF(opts)
	}
	return fmt.Errorf("unsupported input %q (want a photo directory or archive: %s)",
		filepath.Base(opts.Input), strings.Join(ArchiveKinds, ", "))
}

// ZipToGIF reads images from a zip archive and writes an animated GIF.
func ZipToGIF(opts Options) error {
	if err := validateOptions(opts); err != nil {
		return err
	}
	zr, err := zip.OpenReader(opts.Input)
	if err != nil {
		return fmt.Errorf("unreadable zip: %w - re-zip your photos as a standard .zip and try again", err)
	}
	defer zr.Close()

	entries := collectZipImageEntries(zr.File)
	if len(entries) == 0 {
		return noImagesError("zip", summarizeZipNames(zr.File))
	}

	sort.SliceStable(entries, func(i, j int) bool {
		bi := strings.ToLower(entries[i].base)
		bj := strings.ToLower(entries[j].base)
		if bi != bj {
			return bi < bj
		}
		return strings.ToLower(entries[i].name) < strings.ToLower(entries[j].name)
	})

	frames, err := decodeEntriesParallel(opts, len(entries), func(i int) (image.Image, error) {
		return decodeZipImage(entries[i].file)
	})
	if err != nil {
		return err
	}
	if err := checkCtx(opts); err != nil {
		return err
	}
	return writeGIF(opts, frames)
}

// DirToGIF reads images from a directory (recursively) and writes an animated GIF.
func DirToGIF(opts Options) error {
	if err := validateOptions(opts); err != nil {
		return err
	}
	entries, err := collectDirImageEntries(opts.Input)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		st, stErr := summarizeDir(opts.Input)
		if stErr != nil {
			return stErr
		}
		return noImagesError("folder", st)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		bi := strings.ToLower(entries[i].base)
		bj := strings.ToLower(entries[j].base)
		if bi != bj {
			return bi < bj
		}
		return strings.ToLower(entries[i].path) < strings.ToLower(entries[j].path)
	})

	frames, err := decodeEntriesParallel(opts, len(entries), func(i int) (image.Image, error) {
		return decodeFileImage(entries[i].path)
	})
	if err != nil {
		return err
	}
	if err := checkCtx(opts); err != nil {
		return err
	}
	return writeGIF(opts, frames)
}

// HasImages reports whether dir contains at least one supported image file.
func HasImages(dir string) bool {
	entries, err := collectDirImageEntries(dir)
	return err == nil && len(entries) > 0
}

func writeGIF(opts Options, frames []image.Image) error {
	if err := validateOptions(opts); err != nil {
		return err
	}
	if len(frames) == 0 {
		return fmt.Errorf("no usable image frames: files looked like photos but could not be decoded - use real %s images and try again", imageKindsHelp)
	}

	delayCS := (opts.DelayMS + 5) / 10 // GIF delay is in 100ths of a second
	if delayCS < 1 {
		delayCS = 1
	}

	maxWidth := opts.MaxWidth
	if maxWidth == 0 {
		// 0 = use the first sorted frame's native width as the baseline.
		maxWidth = frames[0].Bounds().Dx()
		if maxWidth < 1 {
			maxWidth = 1
		}
	}

	paletted, err := encodeFramesParallel(opts, frames, maxWidth)
	if err != nil {
		return err
	}
	if err := checkCtx(opts); err != nil {
		return err
	}

	delays := make([]int, len(paletted))
	for i := range delays {
		delays[i] = delayCS
	}
	screenW, screenH := gifScreenSize(paletted)
	out := &gif.GIF{
		Image:     paletted,
		Delay:     delays,
		LoopCount: opts.Loop,
		// Logical screen must cover every frame; EncodeAll defaults to the
		// first frame only, which fails when later photos are taller/wider.
		Config: image.Config{Width: screenW, Height: screenH},
	}

	dir := filepath.Dir(opts.Output)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("could not create output folder for %q: %w - pick a writable --output path and try again", opts.Output, err)
	}

	if err := checkCtx(opts); err != nil {
		return err
	}

	report(opts, "writing", 0, 1)
	tmp, err := os.CreateTemp(dir, "giffer-*.gif")
	if err != nil {
		return fmt.Errorf("could not write %q: %w - free disk space or pick another --output path and try again", opts.Output, err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if err := gif.EncodeAll(tmp, out); err != nil {
		return encodeGIFError(err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not write %q: %w - free disk space or pick another --output path and try again", opts.Output, err)
	}

	if err := checkCtx(opts); err != nil {
		return err
	}

	if err := os.Rename(tmpName, opts.Output); err != nil {
		return fmt.Errorf("could not write %q: %w - free disk space or pick another --output path and try again", opts.Output, err)
	}
	cleanup = false
	report(opts, "writing", 1, 1)
	return nil
}

// gifScreenSize returns a logical screen large enough for every frame.
func gifScreenSize(frames []*image.Paletted) (w, h int) {
	for _, p := range frames {
		if p == nil {
			continue
		}
		b := p.Bounds()
		if b.Max.X > w {
			w = b.Max.X
		}
		if b.Max.Y > h {
			h = b.Max.Y
		}
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// encodeGIFError turns low-level gif package errors into actionable guidance.
func encodeGIFError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "out of bounds"):
		return fmt.Errorf("could not encode GIF: photo sizes do not fit one canvas - set --max-width to a shared size (e.g. 800) and try again")
	case strings.Contains(msg, "too large"):
		return fmt.Errorf("photos are too large for GIF (each side must be under 65536px) - lower --max-width and try again")
	default:
		return fmt.Errorf("could not encode GIF (%v) - try a smaller --max-width or fewer photos, then try again", err)
	}
}

type zipImage struct {
	file *zip.File
	name string // zip entry name (full path within archive)
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
		out = append(out, zipImage{file: f, name: name, base: base})
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
	limited := io.LimitReader(r, MaxEntryBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxEntryBytes {
		return nil, fmt.Errorf("image exceeds size limit")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return applyEXIFOrientation(img, data), nil
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
	// Always origin at (0,0) so frames sit inside the GIF logical screen.
	dst := image.NewPaletted(image.Rect(0, 0, b.Dx(), b.Dy()), palette.Plan9)
	draw.FloydSteinberg.Draw(dst, dst.Bounds(), src, b.Min)
	return dst
}
