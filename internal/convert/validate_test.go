package convert

import (
	"archive/zip"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckPhotoSourceRejectsAudioZip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "songs.zip")
	if err := writeRawZip(path, map[string][]byte{
		"track01.mp3": []byte("ID3fake-mp3-bytes"),
		"track02.mp3": []byte("ID3more-fake-bytes"),
		"notes.txt":   []byte("playlist"),
	}); err != nil {
		t.Fatal(err)
	}

	err := CheckPhotoSource(path)
	if err == nil {
		t.Fatal("expected error for audio-only zip")
	}
	msg := err.Error()
	if !strings.Contains(msg, "no supported images") {
		t.Fatalf("want no supported images, got %q", msg)
	}
	if !strings.Contains(msg, "audio") {
		t.Fatalf("want audio hint, got %q", msg)
	}
	if !strings.Contains(msg, "JPEG") {
		t.Fatalf("want corrective photo guidance, got %q", msg)
	}
	if !strings.Contains(msg, "try again") {
		t.Fatalf("want try-again guidance, got %q", msg)
	}
}

func TestZipToGIFRejectsAudioZipWithGuidance(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "music.zip")
	if err := writeRawZip(path, map[string][]byte{
		"a.mp3": []byte("audio"),
		"b.wav": []byte("wave"),
	}); err != nil {
		t.Fatal(err)
	}

	err := ZipToGIF(Options{
		Input:    path,
		Output:   filepath.Join(dir, "out.gif"),
		DelayMS:  100,
		MaxWidth: 800,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := UserMessage(err)
	if !strings.Contains(msg, "audio") {
		t.Fatalf("UserMessage should mention audio, got %q", msg)
	}
	if !strings.Contains(msg, "Zip only photos") {
		t.Fatalf("UserMessage should tell user how to fix, got %q", msg)
	}
}

func TestUserMessageEncodeOutOfBounds(t *testing.T) {
	msg := UserMessage(fmt.Errorf("encode gif: gif: image block is out of bounds"))
	if strings.Contains(msg, "image block is out of bounds") && !strings.Contains(msg, "try again") {
		t.Fatalf("raw gif error without action: %q", msg)
	}
	if !strings.Contains(strings.ToLower(msg), "max-width") && !strings.Contains(msg, "try again") {
		t.Fatalf("want actionable encode guidance, got %q", msg)
	}
}

func TestCheckPhotoSourceAcceptsPhotoZip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shots.zip")
	if err := writeTestZip(path, map[string]imageEntry{
		"a.png": {img: solid(8, 8, color.White), format: "png"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := CheckPhotoSource(path); err != nil {
		t.Fatalf("photo zip should pass: %v", err)
	}
}

func writeRawZip(path string, files map[string][]byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return err
		}
		if _, err := w.Write(data); err != nil {
			_ = zw.Close()
			return err
		}
	}
	return zw.Close()
}
