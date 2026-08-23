package ui_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlienFacepalm/giffer/internal/ui"
)

func TestIndexAndAssets(t *testing.T) {
	srv := ui.New(ui.Options{UploadDir: t.TempDir()})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	for _, path := range []string{"/", "/app.css", "/app.js"} {
		res, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s: status %d", path, res.StatusCode)
		}
		if len(body) == 0 {
			t.Fatalf("%s: empty body", path)
		}
	}
}

func TestConvertAPI(t *testing.T) {
	upload := t.TempDir()
	srv := ui.New(ui.Options{UploadDir: upload})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	zipBytes := mustPNGZip(t, "a.png", 40, 20)
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", "shots.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(zipBytes); err != nil {
		t.Fatal(err)
	}
	_ = w.WriteField("delay-ms", "200")
	_ = w.WriteField("max-width", "20")
	_ = w.WriteField("loop", "2")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	res, err := http.Post(ts.URL+"/api/convert", w.FormDataContentType(), &body)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var got struct {
		OK     bool   `json:"ok"`
		Output string `json:"output"`
		URL    string `json:"url"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || !got.OK {
		t.Fatalf("status=%d ok=%v err=%q", res.StatusCode, got.OK, got.Error)
	}
	if !strings.HasSuffix(got.Output, "shots.gif") {
		t.Fatalf("output=%q", got.Output)
	}
	if _, err := os.Stat(filepath.Join(upload, "shots.gif")); err != nil {
		t.Fatal(err)
	}

	gifRes, err := http.Get(ts.URL + got.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer gifRes.Body.Close()
	if gifRes.StatusCode != http.StatusOK {
		t.Fatalf("gif status %d", gifRes.StatusCode)
	}
	if ct := gifRes.Header.Get("Content-Type"); ct != "image/gif" {
		t.Fatalf("content-type=%q", ct)
	}
}

func TestConvertAPIValidation(t *testing.T) {
	srv := ui.New(ui.Options{UploadDir: t.TempDir()})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	t.Run("missing file", func(t *testing.T) {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		_ = w.WriteField("delay-ms", "500")
		_ = w.Close()
		res, err := http.Post(ts.URL+"/api/convert", w.FormDataContentType(), &body)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("status=%d", res.StatusCode)
		}
	})

	t.Run("bad delay", func(t *testing.T) {
		zipBytes := mustPNGZip(t, "a.png", 10, 10)
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		fw, _ := w.CreateFormFile("file", "x.zip")
		_, _ = fw.Write(zipBytes)
		_ = w.WriteField("delay-ms", "0")
		_ = w.Close()
		res, err := http.Post(ts.URL+"/api/convert", w.FormDataContentType(), &body)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("status=%d", res.StatusCode)
		}
	})
}

func mustPNGZip(t *testing.T, name string, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	wr, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.White)
		}
	}
	if err := png.Encode(wr, img); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
