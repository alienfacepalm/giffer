package main

import (
	"os"

	"github.com/AlienFacepalm/giffer/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
