//go:build windows

package cli

import (
	"io"
	"os"
	"syscall"
)

func guiLaunchPreferred(in io.Reader, out io.Writer) bool {
	// Console attached (terminal) → wizard/batch, never auto-GUI.
	if getConsoleWindow() != 0 {
		return false
	}
	// GUI-subsystem builds (double-click) have no console window.
	// Still reject non-file streams and pipes so tests/CI stay in batch mode
	// (NUL redirects used by Explorer are char devices, not pipes).
	inFile, inOK := in.(*os.File)
	outFile, outOK := out.(*os.File)
	if !inOK || !outOK {
		return false
	}
	if isPipeOrSocket(inFile) || isPipeOrSocket(outFile) {
		return false
	}
	return true
}

func getConsoleWindow() uintptr {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetConsoleWindow")
	r, _, _ := proc.Call()
	return r
}

func isPipeOrSocket(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	m := fi.Mode()
	return m&os.ModeNamedPipe != 0 || m&os.ModeSocket != 0
}
