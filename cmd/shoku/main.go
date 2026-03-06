package main

import (
	"os"

	"github.com/stoneream/shokushitsu/internal/command"
)

func main() {
	if err := command.Execute(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
