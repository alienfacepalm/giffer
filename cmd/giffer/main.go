package main

//go:generate go run ../../scripts/makeico ../..

import (
	"os"

	"github.com/AlienFacepalm/giffer/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
