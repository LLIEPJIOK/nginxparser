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

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file %q: %w", path, err)
	}

	logParser := parser.NewParser()

	info, err := logParser.Parse(f)
	if err != nil {
		return fmt.Errorf("parse file: %w", err)
	}

	fmt.Println(info)

	return nil
}
