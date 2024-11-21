package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/LLIEPJIOK/nginxparser/internal/application/parser"
)

func main() {
	if err := parser.Start(); err != nil {
		slog.Error(fmt.Sprintf("parser.Start(): %s", err))
		os.Exit(1)
	}
}
