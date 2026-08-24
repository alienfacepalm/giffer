package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"
)

const gifferHeader = "X-Giffer"

// openWindowFunc opens the UI in a native window. Overridable in tests.
var openWindowFunc = openDesktopWindow

// SetOpenWindowForTest replaces the window opener. Pass nil to restore.
func SetOpenWindowForTest(fn func(url string) error) {
	if fn == nil {
		openWindowFunc = openDesktopWindow
		return
	}
	openWindowFunc = fn
}

// Run starts the UI on opts.Addr (never remaps to another port), opens a native
// window on the main thread, and blocks until the window closes. If the address
// is already in use by another giffer UI, it reclaims that port — never remaps
// to a different one.
func Run(opts Options, stdout io.Writer) error {
	if strings.TrimSpace(opts.UploadDir) == "" {
		opts.UploadDir = DefaultUploadDir()
	}
	if strings.TrimSpace(opts.Addr) == "" {
		opts.Addr = "127.0.0.1:8765"
	}
	if err := ValidateListenAddr(opts.Addr, opts.AllowRemote); err != nil {
		return err
	}

	url := httpURL(opts.Addr)
	ln, err := net.Listen("tcp", opts.Addr)
	if err != nil {
		if !isAddrInUse(err) {
			return err
		}
		// Port is taken — never bind a different one; reclaim only a prior giffer.
		fmt.Fprintf(stdout, "♻️  giffer ui reclaiming %s\n", opts.Addr)
		if freeErr := freePortFunc(opts.Addr); freeErr != nil {
			return fmt.Errorf("reclaim %s: %w", opts.Addr, freeErr)
		}
		ln, err = net.Listen("tcp", opts.Addr)
		if err != nil {
			return fmt.Errorf("listen after reclaim %s: %w", opts.Addr, err)
		}
	}

	fmt.Fprintf(stdout, "🚀 giffer ui listening on %s\n", url)
	fmt.Fprintf(stdout, "📁 upload dir: %s\n", opts.UploadDir)

	if err := os.MkdirAll(opts.UploadDir, 0o755); err != nil {
		_ = ln.Close()
		return fmt.Errorf("create upload dir: %w", err)
	}

	srv := New(opts)
	srv.opts.Addr = ln.Addr().String()
	srv.server = &http.Server{
		Addr:              srv.opts.Addr,
		Handler:           srv.mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.server.Serve(ln)
	}()

	waitForServer(url)

	// WebView2 (and other native webviews) require the platform message pump on
	// the main thread — never call openWindowFunc from a goroutine.
	windowErr := openWindowFunc(url)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.server.Shutdown(ctx)

	err = <-serveErr
	if windowErr != nil {
		return windowErr
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func waitForServer(url string) {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if Probe(url) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func httpURL(addr string) string {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	return "http://" + addr
}

func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage of each socket address") ||
		strings.Contains(msg, "bind: an attempt was made to access a socket in a way forbidden")
}

// Probe reports whether url looks like a running giffer UI.
func Probe(url string) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, strings.TrimRight(url, "/")+"/", nil)
	if err != nil {
		return false
	}
	res, err := client.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.Header.Get(gifferHeader) == "1"
}
