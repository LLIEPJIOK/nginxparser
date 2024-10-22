package parser

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/es-debug/backend-academy-2024-go-template/internal/domain"
	"golang.org/x/sync/errgroup"
)

func get95p[T ~int | ~string](sl []T) T {
	sort.Slice(sl, func(i, j int) bool {
		return sl[i] < sl[j]
	})

	return sl[95*len(sl)/100]
}

const dataLimit = 3

func dataToFileInfo(parseData data) domain.FileInfo {
	avgResponseSize := parseData.sizeSum / parseData.totalRequests
	responseSize95p := get95p(parseData.sizeSlice)

	frequentURLs := make([]domain.URL, 0, len(parseData.urls))
	for url, quantity := range parseData.urls {
		frequentURLs = append(frequentURLs, domain.NewResource(url, quantity))
	}

	sort.Slice(frequentURLs, func(i, j int) bool {
		return frequentURLs[i].Quantity > frequentURLs[j].Quantity
	})

	urlLimit := min(dataLimit, len(frequentURLs))
	frequentURLs = frequentURLs[:urlLimit]

	frequentStatuses := make([]domain.Status, 0, len(parseData.urls))
	for status, quantity := range parseData.statuses {
		frequentStatuses = append(
			frequentStatuses,
			domain.NewStatus(status, http.StatusText(status), quantity),
		)
	}

	sort.Slice(frequentStatuses, func(i, j int) bool {
		return frequentStatuses[i].Quantity > frequentStatuses[j].Quantity
	})

	statusLimit := min(dataLimit, len(frequentStatuses))
	frequentStatuses = frequentStatuses[:statusLimit]

	return domain.NewFileInfo(
		parseData.totalRequests,
		avgResponseSize,
		responseSize95p,
		frequentURLs,
		frequentStatuses,
	)
}

type Parser struct {
	regex *regexp.Regexp
}

func NewParser() *Parser {
	regex := regexp.MustCompile(
		`^(\S+) - (\S+) \[([^\]]+)\] "(\S+) (\S+) (\S+)" (\d+) (\d+) "([^"]+)" "([^"]*)"$`,
	)

	return &Parser{
		regex: regex,
	}
}

func (p *Parser) lineToLog(line string) (log, error) {
	matches := p.regex.FindStringSubmatch(line)
	if matches == nil {
		return log{}, NewErrRegexp("failed to parse log line with regexp")
	}

	timeLayout := "02/Jan/2006:15:04:05 -0700"

	parsedTime, err := time.Parse(timeLayout, matches[3])
	if err != nil {
		return log{}, fmt.Errorf("failed to parse time: %w", err)
	}

	status, err := strconv.Atoi(matches[7])
	if err != nil {
		return log{}, fmt.Errorf("failed to parse status: %w", err)
	}

	bodyBytesSend, err := strconv.Atoi(matches[8])
	if err != nil {
		return log{}, fmt.Errorf("failed to parse bodyBytesSend: %w", err)
	}

	return log{
		remoteAddress: matches[1],
		remoteUser:    matches[2],
		timeLocal:     parsedTime,
		method:        matches[4],
		url:           matches[5],
		httpVersion:   matches[6],
		status:        status,
		bodyBytesSend: bodyBytesSend,
		referer:       matches[9],
		userAgent:     matches[10],
	}, nil
}

func (p *Parser) processLine(lines <-chan line, parseData *data) error {
	for curLine := range lines {
		logEntry, err := p.lineToLog(curLine.text)
		if err != nil {
			return fmt.Errorf("convert line #%d to log entry: %w", curLine.number, err)
		}

		parseData.processLog(&logEntry)
	}

	return nil
}

func (p *Parser) read(in io.Reader) <-chan line {
	lineNumber := 1
	scan := bufio.NewScanner(in)
	linesChan := make(chan line)

	go func() {
		defer close(linesChan)

		for scan.Scan() {
			text := scan.Text()
			linesChan <- newLine(text, lineNumber)

			lineNumber++
		}
	}()

	return linesChan
}

const numberOfGoroutines = 5

func (p *Parser) Parse(in io.Reader) (domain.FileInfo, error) {
	parseData := newData()
	linesChan := p.read(in)

	eg := errgroup.Group{}
	for range numberOfGoroutines {
		eg.Go(func() error {
			return p.processLine(linesChan, &parseData)
		})
	}

	if err := eg.Wait(); err != nil {
		return domain.FileInfo{}, fmt.Errorf("eg.Wait(): %w", err)
	}

	return dataToFileInfo(parseData), nil
}
