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

// New builds a UI server. UploadDir defaults to "upload".
func New(opts Options) *Server {
	if strings.TrimSpace(opts.UploadDir) == "" {
		opts.UploadDir = "upload"
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
	OK     bool   `json:"ok"`
	Output string `json:"output,omitempty"`
	URL    string `json:"url,omitempty"`
	Error  string `json:"error,omitempty"`
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	const maxUpload = 512 << 20 // 512 MiB
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: "invalid upload: " + err.Error()})
		return
	}

	delayMS, err := atoiDefault(r.FormValue("delay-ms"), 500)
	if err != nil || delayMS <= 0 {
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: "delay-ms must be an integer > 0"})
		return
	}
	maxWidth, err := atoiDefault(r.FormValue("max-width"), 800)
	if err != nil || maxWidth < 1 {
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: "max-width must be an integer >= 1"})
		return
	}
	loop, err := atoiDefault(r.FormValue("loop"), 0)
	if err != nil || loop < 0 {
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: "loop must be an integer >= 0"})
		return
	}

	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: "file is required (zip)"})
		return
	}
	defer file.Close()

	base := filepath.Base(hdr.Filename)
	if !strings.EqualFold(filepath.Ext(base), ".zip") {
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: "file must be a .zip archive"})
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

	zipPath := filepath.Join(s.opts.UploadDir, safe)
	outPath := filepath.Join(s.opts.UploadDir, strings.TrimSuffix(safe, filepath.Ext(safe))+".gif")

	dst, err := os.Create(zipPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, convertResponse{Error: "save zip: " + err.Error()})
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		_ = os.Remove(zipPath)
		writeJSON(w, http.StatusInternalServerError, convertResponse{Error: "save zip: " + err.Error()})
		return
	}
	if err := dst.Close(); err != nil {
		writeJSON(w, http.StatusInternalServerError, convertResponse{Error: "save zip: " + err.Error()})
		return
	}

	if err := convert.Convert(convert.Options{
		Input:    zipPath,
		Output:   outPath,
		DelayMS:  delayMS,
		MaxWidth: maxWidth,
		Loop:     loop,
	}); err != nil {
		writeJSON(w, http.StatusBadRequest, convertResponse{Error: err.Error()})
		return
	}

	name := filepath.Base(outPath)
	writeJSON(w, http.StatusOK, convertResponse{
		OK:     true,
		Output: outPath,
		URL:    "/api/gif/" + name,
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
