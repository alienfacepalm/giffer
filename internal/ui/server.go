package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AlienFacepalm/giffer/internal/convert"
)

// Options configures the local UI server.
type Options struct {
	Addr      string // e.g. "127.0.0.1:8765"
	UploadDir string
}

// Server is the Phase 2 local convert UI.
type Server struct {
	opts   Options
	mux    *http.ServeMux
	server *http.Server
}

// New builds a UI server. UploadDir defaults to DefaultUploadDir().
func New(opts Options) *Server {
	if strings.TrimSpace(opts.UploadDir) == "" {
		opts.UploadDir = DefaultUploadDir()
	}
	if strings.TrimSpace(opts.Addr) == "" {
		opts.Addr = "127.0.0.1:8765"
	}
	s := &Server{opts: opts, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	s.mux.HandleFunc("GET /app.css", s.handleCSS)
	s.mux.HandleFunc("GET /app.js", s.handleJS)
	s.mux.HandleFunc("GET /forge.js", s.handleForge)
	s.mux.HandleFunc("GET /three.min.js", s.handleThree)
	s.mux.HandleFunc("GET /afp-mark.png", s.handleMark)
	s.mux.HandleFunc("GET /fonts/{name}", s.handleFont)
	s.mux.HandleFunc("POST /api/convert", s.handleConvert)
	s.mux.HandleFunc("GET /api/gif/{name}", s.handleGIF)
}

// Handler returns the HTTP handler (for tests).
func (s *Server) Handler() http.Handler { return s.mux }

// ListenAndServe starts the UI server and blocks.
func (s *Server) ListenAndServe() error {
	if err := os.MkdirAll(s.opts.UploadDir, 0o755); err != nil {
		return fmt.Errorf("create upload dir: %w", err)
	}
	s.server = &http.Server{
		Addr:              s.opts.Addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	ln, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return err
	}
	s.opts.Addr = ln.Addr().String()
	return s.server.Serve(ln)
}

// Addr returns the bound address after ListenAndServe starts (or the configured addr).
func (s *Server) Addr() string { return s.opts.Addr }

type convertResponse struct {
	OK     bool   `json:"ok,omitempty"`
	Type   string `json:"type,omitempty"` // "progress" | "done" | "error" (NDJSON stream)
	Stage  string `json:"stage,omitempty"`
	Done   int    `json:"done,omitempty"`
	Total  int    `json:"total,omitempty"`
	Pct    int    `json:"percent,omitempty"`
	Output string `json:"output,omitempty"`
	URL    string `json:"url,omitempty"`
	Error  string `json:"error,omitempty"`
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set(gifferHeader, "1")
	_, _ = w.Write(indexHTML)
}

func (s *Server) handleCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write(appCSS)
}

func (s *Server) handleJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(appJS)
}

func (s *Server) handleForge(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(forgeJS)
}

func (s *Server) handleThree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(threeJS)
}

func (s *Server) handleMark(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(afpMarkPNG)
}

func (s *Server) handleFont(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.PathValue("name"))
	if name == "." || name == "" || strings.Contains(name, "..") || !strings.HasSuffix(name, ".woff2") {
		http.NotFound(w, r)
		return
	}
	data, err := fontFS.ReadFile("static/fonts/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "font/woff2")
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	_, _ = w.Write(data)
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	const maxUpload = 512 << 20 // 512 MiB
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		msg := "Upload failed. Use a photo archive under 512 MB and try again."
		if strings.Contains(strings.ToLower(err.Error()), "too large") || strings.Contains(err.Error(), "MaxBytesReader") {
			msg = "Archive is too large (max 512 MB). Zip fewer or smaller photos and try again."
		}
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: msg})
		return
	}

	delayMS, err := atoiDefault(r.FormValue("delay-ms"), 100)
	if err != nil || delayMS <= 0 {
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: "delay-ms must be an integer > 0"})
		return
	}
	maxWidth, err := atoiDefault(r.FormValue("max-width"), 0)
	if err != nil || maxWidth < 0 {
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: "max-width must be an integer >= 0"})
		return
	}
	loop, err := atoiDefault(r.FormValue("loop"), 0)
	if err != nil || loop < 0 {
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: "loop must be an integer >= 0"})
		return
	}

	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: "file is required (photo archive)"})
		return
	}
	defer file.Close()

	base := filepath.Base(hdr.Filename)
	if !convert.IsArchive(base) {
		writeJSON(w, http.StatusBadRequest, convertResponse{
			Error: "That file is not a photo archive. Upload a zip/tar/7z of photos (" +
				convert.ImageKindsHelp + " images), not a single photo, audio, or other file. Supported archives: " +
				strings.Join(convert.ArchiveKinds, ", "),
		})
		return
	}
	safe := sanitizeBase(base)
	if safe == "" {
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: "invalid file name"})
		return
	}

	if err := os.MkdirAll(s.opts.UploadDir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, convertResponse{Error: "create upload dir: " + err.Error()})
		return
	}

	archivePath := filepath.Join(s.opts.UploadDir, safe)
	outPath := filepath.Join(s.opts.UploadDir, convert.ArchiveStem(safe)+".gif")

	dst, err := os.Create(archivePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, convertResponse{Error: "save archive: " + err.Error()})
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		_ = os.Remove(archivePath)
		writeJSON(w, http.StatusInternalServerError, convertResponse{Error: "save archive: " + err.Error()})
		return
	}
	if err := dst.Close(); err != nil {
		writeJSON(w, http.StatusInternalServerError, convertResponse{Error: "save archive: " + err.Error()})
		return
	}

	if err := convert.CheckPhotoSource(archivePath); err != nil {
		_ = os.Remove(archivePath)
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: convert.UserMessage(err)})
		return
	}

	flusher, canFlush := w.(http.Flusher)
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	emit := func(ev convertResponse) {
		_ = json.NewEncoder(w).Encode(ev)
		if canFlush {
			flusher.Flush()
		}
	}

	err = convert.Convert(convert.Options{
		Input:    archivePath,
		Output:   outPath,
		DelayMS:  delayMS,
		MaxWidth: maxWidth,
		Loop:     loop,
		OnProgress: func(p convert.Progress) {
			emit(convertResponse{
				Type:  "progress",
				Stage: p.Stage,
				Done:  p.Done,
				Total: p.Total,
				Pct:   p.Percent,
			})
		},
	})
	if err != nil {
		emit(convertResponse{Type: "error", Error: convert.UserMessage(err)})
		return
	}

	name := filepath.Base(outPath)
	emit(convertResponse{
		Type:   "done",
		OK:     true,
		Output: outPath,
		URL:    "/api/gif/" + name,
		Pct:    100,
	})
}

func (s *Server) handleGIF(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.PathValue("name"))
	if name == "." || name == "" || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	if !strings.EqualFold(filepath.Ext(name), ".gif") {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.opts.UploadDir, name)
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", name))
	http.ServeContent(w, r, name, time.Time{}, f)
}

func writeJSON(w http.ResponseWriter, status int, body convertResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func atoiDefault(s string, def int) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return def, nil
	}
	return strconv.Atoi(s)
}

func sanitizeBase(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	if name == "." || name == ".." || name == "" {
		return ""
	}
	return name
}
