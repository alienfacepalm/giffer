//go:build !windows

package cli

import (
	"io"
	"os"
)

func guiLaunchPreferred(in io.Reader, out io.Writer) bool {
	inFile, inOK := in.(*os.File)
	outFile, outOK := out.(*os.File)
	if !inOK || !outOK {
		return false
	}
	if fileIsTerminal(inFile) || fileIsTerminal(outFile) {
		return false
	}
	// Named pipes / sockets → scripted/CI → batch mode
	if isPipeOrSocket(inFile) || isPipeOrSocket(outFile) {
		return false
	}
	return true
}

func isPipeOrSocket(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	m := fi.Mode()
	return m&os.ModeNamedPipe != 0 || m&os.ModeSocket != 0
}
