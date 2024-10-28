package parser

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

type cmdFlags struct {
	path     string
	format   string
	output   string
	help     bool
	timeFrom *time.Time
	timeTo   *time.Time
}

func readCMDFlags() (cmdFlags, error) {
	var (
		path   string
		from   string
		to     string
		format string
		output string
		help   bool

		timeFrom *time.Time
		timeTo   *time.Time
	)

	flag.StringVar(&path, "path", "", "path to file")
	flag.StringVar(&path, "p", "", "path to file")

	flag.StringVar(&from, "from", "", "filter by time from")
	flag.StringVar(&from, "f", "", "filter by time from")

	flag.StringVar(&to, "to", "", "filter by time to")
	flag.StringVar(&to, "t", "", "filter by time to")

	flag.StringVar(&format, "format", "md", "output format")
	flag.StringVar(&format, "fmt", "md", "output format")

	flag.StringVar(&output, "output", "", "file for output")
	flag.StringVar(&output, "o", "", "file for output")

	flag.BoolVar(&help, "help", false, "commands info")
	flag.BoolVar(&help, "h", false, "commands info")

	flag.Parse()

	if help {
		return cmdFlags{
			help: true,
		}, nil
	}

	if path == "" {
		return cmdFlags{}, ErrEmptyLogPath{}
	}

	if from != "" {
		tm, err := time.Parse(dataLayout, from)
		if err != nil {
			return cmdFlags{}, fmt.Errorf("parse from date: %w", err)
		}

		timeFrom = &tm
	}

	if from != "" {
		tm, err := time.Parse(dataLayout, from)
		if err != nil {
			return cmdFlags{}, fmt.Errorf("parse to date: %w", err)
		}

		timeTo = &tm
	}

	return cmdFlags{
		path:     path,
		format:   strings.ToLower(format),
		output:   output,
		help:     help,
		timeFrom: timeFrom,
		timeTo:   timeTo,
	}, nil
}
