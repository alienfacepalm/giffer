package convert

import (
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"
)

func TestEncodeFramesParallelPreservesOrder(t *testing.T) {
	frames := make([]image.Image, 12)
	for i := range frames {
		// Distinct solid colors so order is observable via corner pixel.
		c := color.RGBA{R: uint8(i * 20), G: 0, B: 255 - uint8(i*20), A: 255}
		frames[i] = solid(16, 12, c)
	}

	out := t.TempDir()
	path := filepath.Join(out, "ordered.gif")
	if err := writeGIF(Options{
		Output:   path,
		DelayMS:  100,
		MaxWidth: 16,
		Loop:     0,
	}, frames); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Image) != len(frames) {
		t.Fatalf("frames=%d want %d", len(g.Image), len(frames))
	}
	for i, img := range g.Image {
		want := frames[i].At(0, 0)
		got := img.At(0, 0)
		wr, wg, wb, _ := want.RGBA()
		gr, gg, gb, _ := got.RGBA()
		// Palette quantization may nudge channels slightly; require same hue band.
		if absDiff(wr, gr) > 0x3000 || absDiff(wb, gb) > 0x3000 || absDiff(wg, gg) > 0x3000 {
			t.Fatalf("frame %d color mismatch: got %#v want roughly %#v", i, got, want)
		}
	}
}

func TestParallelWorkersCaps(t *testing.T) {
	if got := parallelWorkers(0); got != 1 {
		t.Fatalf("n=0 workers=%d want 1", got)
	}
	if got := parallelWorkers(1); got != 1 {
		t.Fatalf("n=1 workers=%d want 1", got)
	}
	if got := parallelWorkers(100); got < 1 || got > 16 {
		t.Fatalf("n=100 workers=%d want 1..16", got)
	}
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
