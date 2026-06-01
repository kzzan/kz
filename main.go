package main

import (
	"os"

	"github.com/kzzan/kz/pkg/cli"
)

func main() {
	os.Exit(cli.Execute())
}
