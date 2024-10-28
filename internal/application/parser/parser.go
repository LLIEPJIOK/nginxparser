package parser

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/es-debug/backend-academy-2024-go-template/internal/parser"
)

const dataLayout = "2006-01-02"

func Start() error {
	var (
		path string
		from string
		to   string

		timeFrom *time.Time
		timeTo   *time.Time
	)

	flag.StringVar(&path, "path", "", "path to file")
	flag.StringVar(&path, "p", "", "path to file")
	flag.StringVar(&from, "from", "", "filter by time from")
	flag.StringVar(&from, "f", "", "filter by time from")
	flag.StringVar(&to, "to", "", "filter by time to")
	flag.StringVar(&to, "t", "", "filter by time to")

	flag.Parse()

	if path == "" {
		return ErrEmptyLogPath{}
	}

	if from != "" {
		tm, err := time.Parse(dataLayout, from)
		if err != nil {
			return fmt.Errorf("parse from date: %w", err)
		}

		timeFrom = &tm
	}

	if from != "" {
		tm, err := time.Parse(dataLayout, from)
		if err != nil {
			return fmt.Errorf("parse to date: %w", err)
		}

		timeTo = &tm
	}

	logParser := parser.NewParser()

	info, err := logParser.Parse(path, timeFrom, timeTo)
	if err != nil {
		return fmt.Errorf("parse file: %w", err)
	}

	f, _ := os.OpenFile("test.md", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	logParser.Markdown(info, f)

	return nil
}
