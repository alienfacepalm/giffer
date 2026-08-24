//go:build !desktop

package ui

func openDesktopWindow(url string) error {
	return openBrowser(url)
}
