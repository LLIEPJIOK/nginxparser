package parser

import (
	"flag"
	"fmt"
	"os"

	"github.com/es-debug/backend-academy-2024-go-template/internal/parser"
)

func Start() error {
	var path string

	flag.StringVar(&path, "path", "", "path to file")
	flag.StringVar(&path, "p", "", "path to file")

	flag.Parse()

	if path == "" {
		return ErrEmptyLogPath{}
	}

	logParser := parser.NewParser()

	info, err := logParser.Parse(path)
	if err != nil {
		return fmt.Errorf("parse file: %w", err)
	}

	f, _ := os.OpenFile("test.md", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	logParser.Markdown(info, f)

	return nil
}
