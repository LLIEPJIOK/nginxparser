package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/es-debug/backend-academy-2024-go-template/internal/application/parser"
)

func main() {
	if err := parser.Start(); err != nil {
		slog.Error(fmt.Sprintf("parser.Start(): %s", err))
		os.Exit(1)
	}
}
