//go:build desktop && (linux || darwin)

package ui

import webview "github.com/webview/webview_go"

func openDesktopWindow(url string) error {
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("Giffer")
	w.SetSize(1200, 800, webview.HintNone)
	w.Navigate(url)
	w.Run()
	return nil
}
