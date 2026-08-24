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

	for _, path := range []string{
		"/", "/app.css", "/app.js", "/forge.js", "/three.min.js", "/afp-mark.png",
		"/fonts/ibm-plex-mono-400-latin.woff2", "/fonts/syne-latin.woff2",
	} {
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

	indexRes, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	indexBody, _ := io.ReadAll(indexRes.Body)
	indexRes.Body.Close()
	html := string(indexBody)
	if !strings.Contains(html, "alienfacepalm.com") || !strings.Contains(html, "an AlienFacepalm joint") {
		t.Fatal("index missing AlienFacepalm footer credit")
	}
	if !strings.Contains(html, "/afp-mark.png") {
		t.Fatal("index missing outline mark src")
	}
	if !strings.Contains(html, "/three.min.js") || !strings.Contains(html, "/forge.js") {
		t.Fatal("index missing forge animation scripts")
	}
	for _, forbidden := range []string{"fonts.googleapis.com", "fonts.gstatic.com"} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("index must not reference external fonts (%s)", forbidden)
		}
	}
	if !strings.Contains(html, `id="forge"`) {
		t.Fatal("index missing forge mount for generating animation")
	}
	if !strings.Contains(html, `id="reset"`) {
		t.Fatal("index missing Reset control to restore baseline")
	}

	cssRes, err := http.Get(ts.URL + "/app.css")
	if err != nil {
		t.Fatal(err)
	}
	cssBody, _ := io.ReadAll(cssRes.Body)
	cssRes.Body.Close()
	// Preview pane stays mounted; empty state uses .is-empty (not [hidden]).
	if !strings.Contains(string(cssBody), ".result.is-empty") {
		t.Fatal("app.css missing .result.is-empty rule for empty preview pane")
	}
	if !strings.Contains(string(cssBody), "100dvh") {
		t.Fatal("app.css missing single-screen 100dvh layout")
	}
	if !strings.Contains(string(cssBody), `[data-contrast="dark"]`) {
		t.Fatal("app.css missing dark contrast scheme for readable text")
	}
	if !strings.Contains(string(cssBody), `url("/fonts/`) {
		t.Fatal("app.css must load self-hosted fonts")
	}
	if !strings.Contains(html, "data-contrast") {
		t.Fatal("index missing early data-contrast bootstrap")
	}

	jsRes, err := http.Get(ts.URL + "/app.js")
	if err != nil {
		t.Fatal(err)
	}
	jsBody, _ := io.ReadAll(jsRes.Body)
	jsRes.Body.Close()
	if !strings.Contains(string(jsBody), "__gifferContrast") {
		t.Fatal("app.js missing contrast helper export for e2e checks")
	}
	if !strings.Contains(string(jsBody), "resetToBaseline") {
		t.Fatal("app.js missing resetToBaseline for clearing UI state")
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
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "ndjson") {
		t.Fatalf("content-type=%q want ndjson", ct)
	}

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		OK      bool   `json:"ok"`
		Type    string `json:"type"`
		Output  string `json:"output"`
		URL     string `json:"url"`
		Error   string `json:"error"`
		Percent int    `json:"percent"`
	}
	sawProgress := false
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var ev struct {
			OK      bool   `json:"ok"`
			Type    string `json:"type"`
			Output  string `json:"output"`
			URL     string `json:"url"`
			Error   string `json:"error"`
			Percent int    `json:"percent"`
			Stage   string `json:"stage"`
		}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("ndjson line %q: %v", line, err)
		}
		switch ev.Type {
		case "progress":
			sawProgress = true
			if ev.Percent < 0 || ev.Percent > 100 {
				t.Fatalf("percent=%d", ev.Percent)
			}
			// Zero counters must be present in JSON (not omitted) so the UI
			// never interpolates undefined into "Writing GIF (undefined/1)".
			if !strings.Contains(line, `"done"`) || !strings.Contains(line, `"total"`) || !strings.Contains(line, `"percent"`) {
				t.Fatalf("progress event missing numeric fields: %s", line)
			}
		case "done":
			got.OK = ev.OK
			got.Type = ev.Type
			got.Output = ev.Output
			got.URL = ev.URL
			got.Error = ev.Error
			got.Percent = ev.Percent
		case "error":
			t.Fatalf("stream error: %s", ev.Error)
		}
	}
	if !sawProgress {
		t.Fatal("expected progress events in stream")
	}
	if !got.OK || got.Type != "done" {
		t.Fatalf("final=%+v", got)
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

	t.Run("audio zip", func(t *testing.T) {
		var zbuf bytes.Buffer
		zw := zip.NewWriter(&zbuf)
		wr, err := zw.Create("song.mp3")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := wr.Write([]byte("ID3not-a-photo")); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}

		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		fw, err := w.CreateFormFile("file", "songs.zip")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(zbuf.Bytes()); err != nil {
			t.Fatal(err)
		}
		_ = w.WriteField("delay-ms", "100")
		_ = w.WriteField("max-width", "0")
		_ = w.WriteField("loop", "0")
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}

		res, err := http.Post(ts.URL+"/api/convert", w.FormDataContentType(), &body)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", res.StatusCode, readBody(t, res))
		}
		raw := readBody(t, res)
		if !strings.Contains(raw, "audio") {
			t.Fatalf("want audio guidance, got %s", raw)
		}
		if !strings.Contains(raw, "JPEG") && !strings.Contains(raw, "photos") {
			t.Fatalf("want fix-it guidance, got %s", raw)
		}
	})

	t.Run("non archive file", func(t *testing.T) {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		fw, err := w.CreateFormFile("file", "track.mp3")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte("ID3nope")); err != nil {
			t.Fatal(err)
		}
		_ = w.WriteField("delay-ms", "100")
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		res, err := http.Post(ts.URL+"/api/convert", w.FormDataContentType(), &body)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("status=%d", res.StatusCode)
		}
		raw := readBody(t, res)
		if !strings.Contains(raw, "photo archive") {
			t.Fatalf("want photo archive guidance, got %s", raw)
		}
	})
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
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
