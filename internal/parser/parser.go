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

func closeFiles(files []*os.File) {
	for _, f := range files {
		if f != nil {
			closeResource(f)
		}
	}
}

func getFiles(pattern string) ([]*os.File, error) {
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("find files for pattern %q: %w", pattern, err)
	}

	if len(paths) == 0 {
		return nil, NewErrNoFiles("no files for this pattern")
	}

	files := make([]*os.File, len(paths))

	for i, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			closeFiles(files)
			return nil, fmt.Errorf("open file %q: %w", path, err)
		}

		files[i] = f
	}

	return files, nil
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

func (p *Parser) read(ctx context.Context, eg *errgroup.Group, reader io.ReadCloser) <-chan line {
	lines := make(chan line)

	lineNumber := 1
	scan := bufio.NewScanner(reader)

	eg.Go(func() error {
		defer close(lines)

		for scan.Scan() {
			text := scan.Text()
			select {
			case lines <- newLine(text, lineNumber):

			case <-ctx.Done():
				return nil
			}

			lineNumber++
		}

		return nil
	})

	return lines
}

func (p *Parser) parseFilesFanOut(
	ctx context.Context,
	eg *errgroup.Group,
	files []*os.File,
) []<-chan line {
	chs := make([]<-chan line, len(files))

	for i, f := range files {
		chs[i] = p.read(ctx, eg, f)
		files[i] = f
	}

	return chs
}

func (p *Parser) parseFilesFanIn(
	ctx context.Context,
	eg *errgroup.Group,
	chs ...<-chan line,
) <-chan line {
	lines := make(chan line)

	wg := &sync.WaitGroup{}

	for _, ch := range chs {
		wg.Add(1)

		eg.Go(func() error {
			defer wg.Done()

			for l := range ch {
				select {
				case lines <- l:

				case <-ctx.Done():
					return nil
				}
			}

			return nil
		})
	}

	go func() {
		wg.Wait()
		close(lines)
	}()

	return lines
}

func (p *Parser) convertLine(
	ctx context.Context,
	eg *errgroup.Group,
	lines <-chan line,
) <-chan log {
	logs := make(chan log)

	eg.Go(func() error {
		defer close(logs)

		for curLine := range lines {
			logEntry, err := p.lineToLog(curLine.text)
			if err != nil {
				return fmt.Errorf("convert line #%d to log entry: %w", curLine.number, err)
			}

			select {
			case logs <- logEntry:

			case <-ctx.Done():
				return nil
			}
		}

		return nil
	})

	return logs
}

const convertGoroutines = 2

func (p *Parser) convertLineFanOut(
	ctx context.Context,
	eg *errgroup.Group,
	lines <-chan line,
) []<-chan log {
	chs := make([]<-chan log, convertGoroutines)

	for i := range convertGoroutines {
		chs[i] = p.convertLine(ctx, eg, lines)
	}

	return chs
}

func (p *Parser) convertLineFanIn(
	ctx context.Context,
	eg *errgroup.Group,
	chs ...<-chan log,
) <-chan log {
	wg := &sync.WaitGroup{}
	logs := make(chan log)

	for _, ch := range chs {
		wg.Add(1)

		eg.Go(func() error {
			defer wg.Done()

			for lg := range ch {
				select {
				case logs <- lg:

				case <-ctx.Done():
					return nil
				}
			}

			return nil
		})
	}

	go func() {
		wg.Wait()
		close(logs)
	}()

	return logs
}

func (p *Parser) filterTime(
	ctx context.Context,
	eg *errgroup.Group,
	from, to *time.Time,
	filterChan <-chan log,
) <-chan log {
	finalChan := make(chan log)

	eg.Go(func() error {
		defer close(finalChan)

		for lg := range filterChan {
			if from != nil && from.After(lg.timeLocal) || to != nil && to.Before(lg.timeLocal.Truncate(24*time.Hour)) {
				continue
			}

			select {
			case finalChan <- lg:

			case <-ctx.Done():
				return nil
			}
		}

		return nil
	})

	return finalChan
}

const filterTimeGoroutines = 2

func (p *Parser) filterTimeFanOut(
	ctx context.Context,
	eg *errgroup.Group,
	from, to *time.Time,
	filterChan <-chan log,
) []<-chan log {
	chs := make([]<-chan log, filterTimeGoroutines)

	for i := range filterTimeGoroutines {
		chs[i] = p.filterTime(ctx, eg, from, to, filterChan)
	}

	return chs
}

func (p *Parser) filterTimeFanIn(eg *errgroup.Group, chs ...<-chan log) <-chan log {
	wg := &sync.WaitGroup{}
	logs := make(chan log)

	for _, ch := range chs {
		wg.Add(1)

		eg.Go(func() error {
			defer wg.Done()

			for lg := range ch {
				logs <- lg
			}

			return nil
		})
	}

	go func() {
		wg.Wait()
		close(logs)
	}()

	return logs
}

func (p *Parser) collect(
	ctx context.Context,
	eg *errgroup.Group,
	finalChan <-chan log,
	parseData *data,
) {
	eg.Go(func() error {
		for {
			select {
			case lg, ok := <-finalChan:
				if !ok {
					return nil
				}

				parseData.processLog(&lg)

			case <-ctx.Done():
				return nil
			}
		}
	})
}

const collectGoroutines = 2

func (p *Parser) collectFanOut(
	ctx context.Context,
	eg *errgroup.Group,
	collectChan <-chan log,
	parseData *data,
) {
	for range collectGoroutines {
		p.collect(ctx, eg, collectChan, parseData)
	}
}

func (p *Parser) Parse(path string, from, to *time.Time) (*domain.FileInfo, error) {
	var lines <-chan line

	eg, ctx := errgroup.WithContext(context.Background())

	if pathURL, err := parseURL(path); err == nil {
		resp, err := http.Get(pathURL.String())
		if err != nil {
			return nil, fmt.Errorf("get file from url: %w", err)
		}

		defer closeResource(resp.Body)

		lines = p.read(ctx, eg, resp.Body)
	} else {
		slog.Debug(fmt.Sprintf("parse %q as url: %s", path, err))

		files, err := getFiles(path)
		if err != nil {
			return nil, fmt.Errorf("getFiles(%q): %w", path, err)
		}

		defer closeFiles(files)

		lines = p.parseFilesFanIn(ctx, eg, p.parseFilesFanOut(ctx, eg, files)...)
	}

	filterChan := p.convertLineFanIn(ctx, eg, p.convertLineFanOut(ctx, eg, lines)...)
	collectChan := p.filterTimeFanIn(eg, p.filterTimeFanOut(ctx, eg, from, to, filterChan)...)

	parseData := newData()
	p.collectFanOut(ctx, eg, collectChan, &parseData)

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
