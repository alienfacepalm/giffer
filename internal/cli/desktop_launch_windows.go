//go:build windows

package cli

import (
	"io"
	"os"
	"syscall"
)

func guiLaunchPreferred(in io.Reader, out io.Writer) bool {
	inFile, inOK := in.(*os.File)
	outFile, outOK := out.(*os.File)
	if !inOK || !outOK {
		return false
	}
	// GUI-subsystem builds (double-click) have no console window.
	if getConsoleWindow() == 0 {
		return true
	}
	return !fileIsTerminal(inFile) && !fileIsTerminal(outFile)
}

func getConsoleWindow() uintptr {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetConsoleWindow")
	r, _, _ := proc.Call()
	return r
}
