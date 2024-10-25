package parser

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/es-debug/backend-academy-2024-go-template/internal/domain"
	"golang.org/x/sync/errgroup"
)

func parseURL(path string) (*url.URL, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, NewErrInvalidURL(fmt.Sprintf("unsupported scheme: %s", u.Scheme))
	}

	if u.Host == "" {
		return nil, NewErrInvalidURL("host is empty")
	}

	return u, nil
}

func closeResource(res io.Closer) {
	if err := res.Close(); err != nil {
		slog.Error(fmt.Sprintf("close reader: %s", err))
	}
}

func get95p[T ~int | ~string](sl []T) T {
	sort.Slice(sl, func(i, j int) bool {
		return sl[i] < sl[j]
	})

	return sl[95*len(sl)/100]
}

const frequencyLimit = 3

func dataToFileInfo(parseData data) *domain.FileInfo {
	if parseData.totalRequests == 0 {
		return &domain.FileInfo{}
	}

	avgResponseSize := parseData.sizeSum / parseData.totalRequests
	responseSize95p := get95p(parseData.sizeSlice)

	frequentURLs := make([]domain.URL, 0, len(parseData.urls))
	for url, quantity := range parseData.urls {
		frequentURLs = append(frequentURLs, domain.NewURL(url, quantity))
	}

	sort.Slice(frequentURLs, func(i, j int) bool {
		if frequentURLs[i].Quantity != frequentURLs[j].Quantity {
			return frequentURLs[i].Quantity > frequentURLs[j].Quantity
		}

		return frequentURLs[i].Name < frequentURLs[j].Name
	})

	urlLimit := min(frequencyLimit, len(frequentURLs))
	frequentURLs = frequentURLs[:urlLimit]

	frequentStatuses := make([]domain.Status, 0, len(parseData.urls))
	for status, quantity := range parseData.statuses {
		frequentStatuses = append(
			frequentStatuses,
			domain.NewStatus(status, http.StatusText(status), quantity),
		)
	}

	sort.Slice(frequentStatuses, func(i, j int) bool {
		if frequentStatuses[i].Quantity != frequentStatuses[j].Quantity {
			return frequentStatuses[i].Quantity > frequentStatuses[j].Quantity
		}

		return frequentStatuses[i].Code < frequentStatuses[j].Code
	})

	statusLimit := min(frequencyLimit, len(frequentStatuses))
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

	if http.StatusText(status) == "" {
		return log{}, NewErrBadStatus("no such status")
	}

	bodyBytesSent, err := strconv.Atoi(matches[8])
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
		bodyBytesSend: bodyBytesSent,
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

func (p *Parser) read(ctx context.Context, reader io.ReadCloser, linesChan chan<- line) {
	lineNumber := 1
	scan := bufio.NewScanner(reader)

	go func() {
		defer close(linesChan)

		for scan.Scan() {
			text := scan.Text()
			select {
			case linesChan <- newLine(text, lineNumber):

			case <-ctx.Done():
				return
			}

			lineNumber++
		}
	}()
}

func (p *Parser) ParseFiles(ctx context.Context, fileNames []string, linesChan chan<- line) error {
	chs := make([]chan line, len(fileNames))
	files := make([]*os.File, len(fileNames))

	clean := func() {
		for _, file := range files {
			if file != nil {
				closeResource(file)
			}
		}
	}

	for i, fileName := range fileNames {
		f, err := os.Open(fileName)
		if err != nil {
			clean()
			return fmt.Errorf("open file %q: %w", fileName, err)
		}

		chs[i] = make(chan line)
		p.read(ctx, f, chs[i])

		files[i] = f
	}

	wg := &sync.WaitGroup{}

	for _, ch := range chs {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for l := range ch {
				select {
				case linesChan <- l:

				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		clean()
		close(linesChan)
	}()

	return nil
}

func (p *Parser) HandleFilesPath(ctx context.Context, path string, linesChan chan<- line) error {
	fileNames, err := filepath.Glob(path)
	if err != nil {
		return fmt.Errorf("find files for pattern %q: %w", path, err)
	}

	if len(fileNames) == 0 {
		return NewErrNoFiles("no files for this pattern")
	}

	err = p.ParseFiles(ctx, fileNames, linesChan)
	if err != nil {
		return fmt.Errorf("parse files: %w", err)
	}

	return nil
}

const numberOfGoroutines = 5

func (p *Parser) Parse(path string) (*domain.FileInfo, error) {
	linesChan := make(chan line)
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	if pathURL, err := parseURL(path); err == nil {
		resp, err := http.Get(pathURL.String())
		if err != nil {
			return nil, fmt.Errorf("get file from url: %w", err)
		}

		defer closeResource(resp.Body)

		p.read(ctx, resp.Body, linesChan)
	} else {
		slog.Debug(fmt.Sprintf("parse %q as url: %s", path, err))

		if err := p.HandleFilesPath(ctx, path, linesChan); err != nil {
			return nil, fmt.Errorf("handle files path: %w", err)
		}
	}

	parseData := newData()

	eg := errgroup.Group{}
	for range numberOfGoroutines {
		eg.Go(func() error {
			return p.processLine(linesChan, &parseData)
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("eg.Wait(): %w", err)
	}

	fileInfo := dataToFileInfo(parseData)
	fileInfo.Path = path

	return fileInfo, nil
}

func (p *Parser) Markdown(info *domain.FileInfo, out io.Writer) {
	fmt.Fprint(out, "#### General information\n\n")
	fmt.Fprint(out, "| Метрика | Значение |\n")
	fmt.Fprint(out, "|:-|-:|\n")
	fmt.Fprintf(out, "| File | %s |\n", info.Path)
	fmt.Fprintf(out, "| Number of requests | %d |\n", info.TotalRequests)
	fmt.Fprintf(out, "| Average Response Size | %d |\n", info.AvgResponseSize)
	fmt.Fprintf(out, "| 95th Percentile of Response Size | %d |\n\n", info.ResponseSize95p)

	fmt.Fprint(out, "#### Requested resources\n\n")
	fmt.Fprint(out, "| Resource | Count |\n")
	fmt.Fprint(out, "|:-|-:|\n")

	for _, url := range info.FrequentURLs {
		fmt.Fprintf(out, "| `%s` | %d |\n", url.Name, url.Quantity)
	}

	fmt.Fprint(out, "#### Response codes\n\n")
	fmt.Fprint(out, "| Code | Name | Count |\n")
	fmt.Fprint(out, "|:-|:-:|-:|\n")

	for _, status := range info.FrequentStatuses {
		fmt.Fprintf(out, "| %d | %s | %d |\n", status.Code, status.Name, status.Quantity)
	}
}
