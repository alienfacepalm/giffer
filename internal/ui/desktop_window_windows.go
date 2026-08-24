//go:build desktop && windows

package ui

import (
	"github.com/jchv/go-webview2"
)

func openDesktopWindow(url string) error {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		WindowOptions: webview2.WindowOptions{
			Title:  "Giffer",
			Width:  1200,
			Height: 800,
			Center: true,
		},
	})
	if w == nil {
		return openBrowser(url)
	}
	defer w.Destroy()
	w.Navigate(url)
	w.Run()
	return nil
}
