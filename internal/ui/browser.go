package ui

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openBrowserFunc opens url in the default browser. Overridable in tests.
var openBrowserFunc = openBrowser

// SetOpenBrowserForTest replaces the browser opener. Pass nil to restore.
func SetOpenBrowserForTest(fn func(url string) error) {
	if fn == nil {
		openBrowserFunc = openBrowser
		return
	}
	openBrowserFunc = fn
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Empty title arg is required so start treats the URL as the target.
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
