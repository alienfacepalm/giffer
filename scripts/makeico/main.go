// Command makeico builds assets/giffer.ico (and Windows .syso resources)
// from assets/giffer-icon.png.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/image/draw"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	if err := generateSourcePNG(root); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not regenerate giffer-icon.png: %v\n", err)
	}

	pngPath := filepath.Join(root, "assets", "giffer-icon.png")
	icoPath := filepath.Join(root, "assets", "giffer.ico")

	f, err := os.Open(pngPath)
	if err != nil {
		fail(err)
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		fail(err)
	}

	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	var entries []icoEntry
	for _, s := range sizes {
		dst := image.NewRGBA(image.Rect(0, 0, s, s))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
		pngBytes, err := encodePNG(dst)
		if err != nil {
			fail(err)
		}
		entries = append(entries, icoEntry{size: s, png: pngBytes})
	}
	if err := writeICO(icoPath, entries); err != nil {
		fail(err)
	}
	fmt.Printf("wrote %s (%d bytes)\n", icoPath, fileSize(icoPath))

	if err := embedWindowsIcon(root, icoPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: syso embed skipped: %v\n", err)
		fmt.Fprintf(os.Stderr, "install rsrc: go install github.com/akavel/rsrc@latest\n")
	}
}

func generateSourcePNG(root string) error {
	script := filepath.Join(root, "scripts", "makeico", "gen_icon.py")
	if _, err := os.Stat(script); err != nil {
		return err
	}
	py := "python"
	if _, err := exec.LookPath(py); err != nil {
		py = "python3"
	}
	cmd := exec.Command(py, script)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type icoEntry struct {
	size int
	png  []byte
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeICO(path string, entries []icoEntry) error {
	var buf bytes.Buffer
	// ICONDIR
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0)) // reserved
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1)) // type: icon
	_ = binary.Write(&buf, binary.LittleEndian, uint16(len(entries)))

	offset := 6 + 16*len(entries)
	type dir struct {
		w, h, colors, reserved uint8
		planes, bitCount       uint16
		bytesInRes             uint32
		imageOffset            uint32
	}
	dirs := make([]dir, len(entries))
	for i, e := range entries {
		w := uint8(e.size)
		h := uint8(e.size)
		if e.size >= 256 {
			w, h = 0, 0 // 256 is stored as 0 in ICO
		}
		dirs[i] = dir{
			w: w, h: h, colors: 0, reserved: 0,
			planes: 1, bitCount: 32,
			bytesInRes:  uint32(len(e.png)),
			imageOffset: uint32(offset),
		}
		offset += len(e.png)
	}
	for _, d := range dirs {
		_ = binary.Write(&buf, binary.LittleEndian, d.w)
		_ = binary.Write(&buf, binary.LittleEndian, d.h)
		_ = binary.Write(&buf, binary.LittleEndian, d.colors)
		_ = binary.Write(&buf, binary.LittleEndian, d.reserved)
		_ = binary.Write(&buf, binary.LittleEndian, d.planes)
		_ = binary.Write(&buf, binary.LittleEndian, d.bitCount)
		_ = binary.Write(&buf, binary.LittleEndian, d.bytesInRes)
		_ = binary.Write(&buf, binary.LittleEndian, d.imageOffset)
	}
	for _, e := range entries {
		buf.Write(e.png)
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func embedWindowsIcon(root, icoPath string) error {
	rsrc, err := exec.LookPath("rsrc")
	if err != nil {
		gopath, _ := exec.Command("go", "env", "GOPATH").Output()
		candidate := filepath.Join(string(bytes.TrimSpace(gopath)), "bin", "rsrc.exe")
		if st, e := os.Stat(candidate); e == nil && !st.IsDir() {
			rsrc = candidate
		} else {
			return err
		}
	}
	outDir := filepath.Join(root, "cmd", "giffer")
	for _, arch := range []string{"amd64", "386"} {
		out := filepath.Join(outDir, fmt.Sprintf("rsrc_windows_%s.syso", arch))
		cmd := exec.Command(rsrc, "-arch", arch, "-ico", icoPath, "-o", out)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", arch, err)
		}
		fmt.Printf("wrote %s\n", out)
	}
	return nil
}

func fileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "makeico: %v\n", err)
	os.Exit(1)
}
