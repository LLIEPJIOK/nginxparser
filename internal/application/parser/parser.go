package parser

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/es-debug/backend-academy-2024-go-template/internal/parser"
)

const possibleFilterFields = `
Possible fields for filtration: 
  - RemoteAddress
  - RemoteUser
  - TimeLocal
  - Method
  - Url
  - HTTPVersion
  - Status
  - BodyBytesSend
  - Referer
  - UserAgent

`

func Start() error {
	fl, err := readCMDFlags()
	if err != nil {
		flag.Usage()
		fmt.Print(possibleFilterFields)

		return fmt.Errorf("readCMDFlags(): %w", err)
	}

	if fl.help {
		flag.Usage()
		fmt.Print(possibleFilterFields)

		return nil
	}

	logParser := parser.NewParser()

	info, err := logParser.Parse(parser.Params{
		Path:        fl.path,
		From:        fl.timeFrom,
		To:          fl.timeTo,
		FilterField: fl.filterField,
		FilterValue: fl.filterValue,
	})
	if err != nil {
		return fmt.Errorf("parse file: %w", err)
	}

	var wr io.Writer

	if fl.output != "" {
		f, err := os.OpenFile(fl.output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("open file %q: %w", fl.output, err)
		}

		defer func() {
			if err := f.Close(); err != nil {
				slog.Error(fmt.Sprintf("close file %q: %s", fl.output, err))
			}
		}()

		wr = f
	} else {
		wr = os.Stdout
	}

	switch fl.format {
	case "adoc":
		logParser.Adoc(info, wr)

	case "md", "markdown":
		logParser.Markdown(info, wr)

	default:
		flag.Usage()
		return NewErrFlag("format: unknown flag")
	}

	return nil
}
