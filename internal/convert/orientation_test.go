package convert

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestTransformOrientation6Rotates90CW(t *testing.T) {
	// 2x1 image: left red, right blue. After orient 6 (90° CW) → 1x2: top red, bottom blue.
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.RGBA{255, 0, 0, 255})
	src.Set(1, 0, color.RGBA{0, 0, 255, 255})

	got := transformOrientation(src, 6)
	b := got.Bounds()
	if b.Dx() != 1 || b.Dy() != 2 {
		t.Fatalf("bounds=%v want 1x2", b)
	}
	if !sameRGB(got.At(0, 0), color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("top=%v want red", got.At(0, 0))
	}
	if !sameRGB(got.At(0, 1), color.RGBA{0, 0, 255, 255}) {
		t.Fatalf("bottom=%v want blue", got.At(0, 1))
	}
}

func TestTransformOrientation8Rotates90CCW(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.RGBA{255, 0, 0, 255})
	src.Set(1, 0, color.RGBA{0, 0, 255, 255})

	got := transformOrientation(src, 8)
	b := got.Bounds()
	if b.Dx() != 1 || b.Dy() != 2 {
		t.Fatalf("bounds=%v want 1x2", b)
	}
	if !sameRGB(got.At(0, 0), color.RGBA{0, 0, 255, 255}) {
		t.Fatalf("top=%v want blue", got.At(0, 0))
	}
	if !sameRGB(got.At(0, 1), color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("bottom=%v want red", got.At(0, 1))
	}
}

func TestDecodeImageAppliesJPEGEXIFOrientation(t *testing.T) {
	// Pixel data is 2x1 landscape; EXIF 6 means display as portrait (90° CW).
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.RGBA{255, 0, 0, 255})
	src.Set(1, 0, color.RGBA{0, 255, 0, 255})

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	jpegWithEXIF, err := injectJPEGOrientation(buf.Bytes(), 6)
	if err != nil {
		t.Fatal(err)
	}

	img, err := decodeImage(bytes.NewReader(jpegWithEXIF))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != 1 || b.Dy() != 2 {
		t.Fatalf("after EXIF orient bounds=%v want 1x2", b)
	}
}

func sameRGB(c color.Color, want color.RGBA) bool {
	r, g, b, _ := c.RGBA()
	return uint8(r>>8) == want.R && uint8(g>>8) == want.G && uint8(b>>8) == want.B
}

// injectJPEGOrientation wraps a baseline JPEG with an APP1 EXIF segment carrying Orientation.
func injectJPEGOrientation(jpegData []byte, orient int) ([]byte, error) {
	if len(jpegData) < 2 || jpegData[0] != 0xff || jpegData[1] != 0xd8 {
		return nil, errNotJPEG
	}
	exif := buildEXIFOrientationAPP1(orient)
	out := make([]byte, 0, 2+len(exif)+len(jpegData)-2)
	out = append(out, 0xff, 0xd8)
	out = append(out, exif...)
	out = append(out, jpegData[2:]...)
	return out, nil
}

var errNotJPEG = errString("not a jpeg")

type errString string

func (e errString) Error() string { return string(e) }

func buildEXIFOrientationAPP1(orient int) []byte {
	// Minimal TIFF/EXIF with a single Orientation short tag (0x0112).
	tiff := []byte{
		'I', 'I', // little-endian
		0x2a, 0x00, // magic
		0x08, 0x00, 0x00, 0x00, // IFD0 offset
		0x01, 0x00, // 1 entry
		0x12, 0x01, // tag Orientation
		0x03, 0x00, // type SHORT
		0x01, 0x00, 0x00, 0x00, // count
		byte(orient), 0x00, 0x00, 0x00, // value
		0x00, 0x00, 0x00, 0x00, // next IFD
	}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	size := len(payload) + 2
	app1 := []byte{0xff, 0xe1, byte(size >> 8), byte(size)}
	return append(app1, payload...)
}
