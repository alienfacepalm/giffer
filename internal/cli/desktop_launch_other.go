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
	return !fileIsTerminal(inFile) && !fileIsTerminal(outFile)
}
