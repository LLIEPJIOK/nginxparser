package parser

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

const dataLayout = "2006-01-02"

type cmdFlags struct {
	path     string
	format   string
	output   string
	help     bool
	timeFrom *time.Time
	timeTo   *time.Time

	filterField string
	filterValue string
}

func parseTime(timeStr string) (*time.Time, error) {
	if timeStr != "" {
		tm, err := time.Parse(dataLayout, timeStr)
		if err != nil {
			return nil, fmt.Errorf("parse time: %w", err)
		}

		return &tm, nil
	}

	return nil, nil
}

func readCMDFlags() (cmdFlags, error) {
	var (
		path   string
		from   string
		to     string
		format string
		output string
		help   bool

		filterField string
		filterValue string

		timeFrom *time.Time
		timeTo   *time.Time

		err error
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

	flag.StringVar(&filterField, "filter-field", "", "field for filtration")
	flag.StringVar(&filterValue, "filter-value", "", "value for filtration")

	flag.Parse()

	if help {
		return cmdFlags{help: true}, nil
	}

	if path == "" {
		return cmdFlags{}, ErrEmptyLogPath{}
	}

	timeFrom, err = parseTime(from)
	if err != nil {
		return cmdFlags{}, fmt.Errorf("parse time from %q: %w", from, err)
	}

	timeTo, err = parseTime(to)
	if err != nil {
		return cmdFlags{}, fmt.Errorf("parse time to %q: %w", to, err)
	}

	return cmdFlags{
		path:        path,
		format:      strings.ToLower(format),
		output:      output,
		help:        help,
		timeFrom:    timeFrom,
		timeTo:      timeTo,
		filterField: filterField,
		filterValue: filterValue,
	}, nil
}
