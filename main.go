package main

import (
	"os"

	"github.com/altfins-com/altfins-cli/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
