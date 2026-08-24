//go:build desktop && windows

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jchv/go-webview2"
)

func openDesktopWindow(url string) error {
	dataPath := webViewDataPath()
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		DataPath: dataPath,
		WindowOptions: webview2.WindowOptions{
			Title:  "Giffer",
			Width:  1200,
			Height: 800,
			Center: true,
		},
	})
	if w == nil {
		_ = appendDesktopLog(fmt.Sprintf("WebView2 window failed to initialize (dataPath=%q)", dataPath))
		return fmt.Errorf("WebView2 failed to initialize — see %s", desktopLogPath())
	}
	defer w.Destroy()
	w.Navigate(url)
	w.Run()
	return nil
}

func webViewDataPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	if isGoBuildBinary(dir) {
		if wd, err := os.Getwd(); err == nil {
			dir = wd
		}
	}
	return filepath.Join(dir, ".giffer-webview2")
}

func appendDesktopLog(msg string) error {
	logPath := desktopLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil && filepath.Dir(logPath) != "." {
		return err
	}
	line := fmt.Sprintf("%s %s\n", time.Now().Format(time.RFC3339), msg)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

func desktopLogPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "giffer.log"
	}
	dir := filepath.Dir(exe)
	if isGoBuildBinary(dir) {
		return "giffer.log"
	}
	return filepath.Join(dir, "giffer.log")
}
