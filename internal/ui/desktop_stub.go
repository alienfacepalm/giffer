//go:build !desktop

package ui

import "fmt"

func openDesktopWindow(url string) error {
	_ = url
	return fmt.Errorf("native UI requires a desktop build (rebuild with -tags desktop)")
}
