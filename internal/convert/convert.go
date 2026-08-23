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

// Options controls a conversion run from one or more inputs.
type Options struct {
	Inputs   []string
	Output   string
	DelayMS  int
	MaxWidth int
	Loop     int
}

// ToGIF reads images from zips, image files, and/or directories and writes an animated GIF.
func ToGIF(opts Options) error {
	if len(opts.Inputs) == 0 {
		return fmt.Errorf("no inputs provided")
	}

	col := &collector{}
	defer col.Close()

	for _, in := range opts.Inputs {
		if err := col.add(in); err != nil {
			return err
		}
	}
	if len(col.refs) == 0 {
		return fmt.Errorf("no supported images found in inputs")
	}

	sort.SliceStable(col.refs, func(i, j int) bool {
		li := strings.ToLower(col.refs[i].base)
		lj := strings.ToLower(col.refs[j].base)
		if li != lj {
			return li < lj
		}
		return col.refs[i].key < col.refs[j].key
	})

	delayCS := (opts.DelayMS + 5) / 10 // GIF delay is in 100ths of a second
	if delayCS < 1 {
		delayCS = 1
	}

	out := &gif.GIF{LoopCount: opts.Loop}
	var frames []*image.Paletted
	for _, ref := range col.refs {
		img, err := ref.open()
		if err != nil {
			continue // skip unreadable frames; require at least one usable frame below
		}
		resized := resizeMaxWidth(img, opts.MaxWidth)
		frames = append(frames, quantize(resized))
	}

	if len(frames) == 0 {
		return fmt.Errorf("no usable image frames")
	}

	width, height := 0, 0
	for _, fr := range frames {
		if w := fr.Bounds().Dx(); w > width {
			width = w
		}
		if h := fr.Bounds().Dy(); h > height {
			height = h
		}
	}
	out.Config = image.Config{Width: width, Height: height}
	for _, fr := range frames {
		out.Image = append(out.Image, fitCanvas(fr, width, height))
		out.Delay = append(out.Delay, delayCS)
	}

	dir := filepath.Dir(opts.Output)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
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

type frameRef struct {
	base string
	key  string
	open func() (image.Image, error)
}

type collector struct {
	zips []*zip.ReadCloser
	refs []frameRef
}

func (c *collector) Close() {
	for _, z := range c.zips {
		_ = z.Close()
	}
}

func (c *collector) add(input string) error {
	info, err := os.Stat(input)
	if err != nil {
		if strings.EqualFold(filepath.Ext(input), ".zip") {
			return fmt.Errorf("unreadable zip: %w", err)
		}
		return fmt.Errorf("unreadable input %q: %w", input, err)
	}

	switch {
	case info.IsDir():
		return c.addDir(input)
	case strings.EqualFold(filepath.Ext(input), ".zip"):
		return c.addZip(input)
	case isSupportedImage(filepath.Base(input)):
		c.addFile(input)
		return nil
	default:
		return fmt.Errorf("unsupported input %q: use a .zip, image file, or directory", input)
	}
}

func (c *collector) addZip(path string) error {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("unreadable zip: %w", err)
	}
	c.zips = append(c.zips, zr)

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := f.Name
		base := pathBase(name)
		if shouldSkipPath(name, base) || !isSupportedImage(base) {
			continue
		}
		zf := f
		c.refs = append(c.refs, frameRef{
			base: base,
			key:  path + "\x00" + name,
			open: func() (image.Image, error) {
				return decodeReader(zf.Open)
			},
		})
	}
	return nil
}

func (c *collector) addDir(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			rel = p
		}
		slashRel := filepath.ToSlash(rel)
		base := filepath.Base(p)

		if d.IsDir() {
			if base == "__MACOSX" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipPath(slashRel, base) || !isSupportedImage(base) {
			return nil
		}
		c.addFile(p)
		return nil
	})
}

func (c *collector) addFile(path string) {
	base := filepath.Base(path)
	p := path
	c.refs = append(c.refs, frameRef{
		base: base,
		key:  p,
		open: func() (image.Image, error) {
			return decodeReader(func() (io.ReadCloser, error) {
				return os.Open(p)
			})
		},
	})
}

func decodeReader(open func() (io.ReadCloser, error)) (image.Image, error) {
	rc, err := open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	img, _, err := image.Decode(rc)
	return img, err
}

func pathBase(name string) string {
	return path.Base(strings.ReplaceAll(name, "\\", "/"))
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
	switch strings.ToLower(path.Ext(filepath.ToSlash(base))) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return true
	default:
		return false
	}
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

func fitCanvas(src *image.Paletted, width, height int) *image.Paletted {
	if src.Bounds().Dx() == width && src.Bounds().Dy() == height && src.Bounds().Min.Eq(image.Pt(0, 0)) {
		return src
	}
	dst := image.NewPaletted(image.Rect(0, 0, width, height), src.Palette)
	draw.Draw(dst, src.Bounds(), src, src.Bounds().Min, draw.Src)
	return dst
}
