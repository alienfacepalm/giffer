//go:build windows

package cli

import "golang.org/x/sys/windows"

func enableWindowsANSI() {
	enableVT(windows.Stdout)
	enableVT(windows.Stderr)
}

func enableVT(handle windows.Handle) {
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return
	}
	_ = windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
