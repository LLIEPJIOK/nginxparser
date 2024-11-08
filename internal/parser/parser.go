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
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
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

func getFiles(paths []string) ([]*os.File, error) {
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

func get95p[T ~int](sl []T) T {
	sort.Slice(sl, func(i, j int) bool {
		return sl[i] < sl[j]
	})

	return sl[95*len(sl)/100]
}

const frequencyLimit = 3

func frequentURLs(parseData *data) []domain.URL {
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

	return frequentURLs
}

func frequentStatuses(parseData *data) []domain.Status {
	frequentStatuses := make([]domain.Status, 0, len(parseData.urls))
	for status, quantity := range parseData.statuses {
		frequentStatuses = append(
			frequentStatuses,
			domain.NewStatus(status, quantity),
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

	return frequentStatuses
}

func frequentAddresses(parseData *data) []domain.Address {
	frequentAddresses := make([]domain.Address, 0, len(parseData.addresses))
	for ip, quantity := range parseData.addresses {
		frequentAddresses = append(
			frequentAddresses,
			domain.NewAddress(ip, quantity),
		)
	}

	sort.Slice(frequentAddresses, func(i, j int) bool {
		if frequentAddresses[i].Quantity != frequentAddresses[j].Quantity {
			return frequentAddresses[i].Quantity > frequentAddresses[j].Quantity
		}

		return frequentAddresses[i].Name < frequentAddresses[j].Name
	})

	addressLimit := min(frequencyLimit, len(frequentAddresses))
	frequentAddresses = frequentAddresses[:addressLimit]

	return frequentAddresses
}

func dataToFileInfo(parseData *data) *domain.FileInfo {
	if parseData.totalRequests == 0 {
		return &domain.FileInfo{}
	}

	avgResponseSize := parseData.sizeSum / parseData.totalRequests
	responseSize95p := get95p(parseData.sizeSlice)

	freqURLs := frequentURLs(parseData)
	freqStatuses := frequentStatuses(parseData)
	freqAddresses := frequentAddresses(parseData)

	avgResponsesPerDay := 0
	for _, quantity := range parseData.requestsPerDay {
		avgResponsesPerDay += quantity
	}

	avgResponsesPerDay /= len(parseData.requestsPerDay)

	return domain.NewFileInfo(
		parseData.paths,
		parseData.totalRequests,
		avgResponseSize,
		responseSize95p,
		avgResponsesPerDay,
		freqURLs,
		freqStatuses,
		freqAddresses,
	)
}

type Parser struct {
	regex      *regexp.Regexp
	timeLayout string
}

func NewParser() *Parser {
	regex := regexp.MustCompile(
		`^(\S+) - (\S+) \[([^\]]+)\] "(\S+) (\S+) (\S+)" (\d+) (\d+) "([^"]+)" "([^"]*)"$`,
	)

	return &Parser{
		regex:      regex,
		timeLayout: "02/Jan/2006:15:04:05 -0700",
	}
}

func (p *Parser) lineToLog(line string) (log, error) {
	matches := p.regex.FindStringSubmatch(line)
	if matches == nil {
		return log{}, NewErrRegexp("failed to parse log line with regexp")
	}

	parsedTime, err := time.Parse(p.timeLayout, matches[3])
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
		RemoteAddress: matches[1],
		RemoteUser:    matches[2],
		TimeLocal:     parsedTime,
		Method:        matches[4],
		URL:           matches[5],
		HTTPVersion:   matches[6],
		Status:        status,
		BodyBytesSend: bodyBytesSent,
		Referer:       matches[9],
		UserAgent:     matches[10],
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
			if from != nil && from.After(lg.TimeLocal) ||
				to != nil && to.Before(lg.TimeLocal.Truncate(24*time.Hour)) {
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

func (p *Parser) filterTimeFanIn(
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

func matchLogByField(logEntry *log, filed, pattern string) (bool, error) {
	fieldValue := reflect.ValueOf(*logEntry).FieldByName(filed)

	if !fieldValue.IsValid() {
		return true, nil
	}

	var value string

	switch fieldValue.Kind() {
	case reflect.String:
		value = fieldValue.String()

	case reflect.Int:
		value = fmt.Sprintf("%d", fieldValue.Int())

	case reflect.Struct:
		if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
			tm := fieldValue.Interface().(time.Time)
			value = tm.Format(timeLayout)
		}

	case reflect.Invalid,
		reflect.Bool,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice,
		reflect.UnsafePointer:
		return false, nil
	}

	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return false, fmt.Errorf("error matching regex: %w", err)
	}

	fmt.Println(pattern, value, matched)

	return matched, nil
}

func (p *Parser) filterField(
	ctx context.Context,
	eg *errgroup.Group,
	field, value string,
	filterChan <-chan log,
) <-chan log {
	finalChan := make(chan log)

	eg.Go(func() error {
		defer close(finalChan)

		for lg := range filterChan {
			match, err := matchLogByField(&lg, field, value)
			if err != nil {
				return fmt.Errorf("matching log by field=%q with value = %q: %w", field, value, err)
			}

			if match {
				select {
				case finalChan <- lg:

				case <-ctx.Done():
					return nil
				}
			}
		}

		return nil
	})

	return finalChan
}

const filterFieldGoroutines = 2

func (p *Parser) filterFieldFanOut(
	ctx context.Context,
	eg *errgroup.Group,
	field, value string,
	filterChan <-chan log,
) []<-chan log {
	chs := make([]<-chan log, filterFieldGoroutines)

	for i := range filterTimeGoroutines {
		chs[i] = p.filterField(ctx, eg, field, value, filterChan)
	}

	return chs
}

func (p *Parser) filterFieldFanIn(
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

func (p *Parser) Parse(prm Params) (*domain.FileInfo, error) {
	var lines <-chan line

	parseData := newData()
	eg, ctx := errgroup.WithContext(context.Background())

	if pathURL, err := parseURL(prm.Path); err == nil {
		resp, err := http.Get(pathURL.String())
		if err != nil {
			return nil, fmt.Errorf("get file from url: %w", err)
		}

		defer closeResource(resp.Body)

		lines = p.read(ctx, eg, resp.Body)
	} else {
		slog.Debug(fmt.Sprintf("parse %q as url: %s", prm.Path, err))

		paths, err := filepath.Glob(prm.Path)
		if err != nil {
			return nil, fmt.Errorf("find files for pattern %q: %w", prm.Path, err)
		}

		parseData.paths = paths

		files, err := getFiles(paths)
		if err != nil {
			return nil, fmt.Errorf("getFiles(%q): %w", prm.Path, err)
		}

		defer closeFiles(files)

		lines = p.parseFilesFanIn(ctx, eg, p.parseFilesFanOut(ctx, eg, files)...)
	}

	filterTimeChan := p.convertLineFanIn(ctx, eg, p.convertLineFanOut(ctx, eg, lines)...)
	filterFieldChan := p.filterFieldFanIn(
		ctx,
		eg,
		p.filterFieldFanOut(ctx, eg, prm.FilterField, prm.FilterValue, filterTimeChan)...)
	collectChan := p.filterTimeFanIn(
		ctx,
		eg,
		p.filterTimeFanOut(ctx, eg, prm.From, prm.To, filterFieldChan)...)
	p.collectFanOut(ctx, eg, collectChan, &parseData)

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("eg.Wait(): %w", err)
	}

	fileInfo := dataToFileInfo(&parseData)

	return fileInfo, nil
}

func (p *Parser) Markdown(info *domain.FileInfo, out io.Writer) {
	fmt.Fprint(out, "#### General information\n\n")
	fmt.Fprint(out, "| Метрика | Значение |\n")
	fmt.Fprint(out, "|:-|-:|\n")
	fmt.Fprintf(out, "| Files | %s |\n", strings.Join(info.Paths, ", "))
	fmt.Fprintf(out, "| Number of requests | %d |\n", info.TotalRequests)
	fmt.Fprintf(out, "| Average response size | %d |\n", info.AvgResponseSize)
	fmt.Fprintf(out, "| 95th Percentile of response size | %d |\n", info.ResponseSize95p)
	fmt.Fprintf(out, "| Average requests per day | %d |\n\n", info.AvgResponsePerDay)

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

	fmt.Fprint(out, "#### Requesting addresses\n\n")
	fmt.Fprint(out, "| Address | Count |\n")
	fmt.Fprint(out, "|:-|-:|\n")

	for _, address := range info.FrequentAddresses {
		fmt.Fprintf(out, "| `%s` | %d |\n", address.Name, address.Quantity)
	}
}

func (p *Parser) Adoc(info *domain.FileInfo, out io.Writer) {
	fmt.Fprint(out, "==== General Information\n\n")
	fmt.Fprint(out, "[options=\"header\"]\n")
	fmt.Fprint(out, "|===\n")
	fmt.Fprint(out, "| Метрика | Значение\n")

	fmt.Fprintf(out, "| Files | %s\n", strings.Join(info.Paths, ", "))
	fmt.Fprintf(out, "| Number of requests | %d\n", info.TotalRequests)
	fmt.Fprintf(out, "| Average response size | %d\n", info.AvgResponseSize)
	fmt.Fprintf(out, "| 95th percentile of response size | %d\n", info.ResponseSize95p)
	fmt.Fprintf(out, "| Average requests per day | %d |\n", info.AvgResponsePerDay)
	fmt.Fprint(out, "|===\n\n")

	fmt.Fprint(out, "==== Requested Resources\n\n")
	fmt.Fprint(out, "[options=\"header\"]\n")
	fmt.Fprint(out, "|===\n")
	fmt.Fprint(out, "| Resource | Count\n")

	for _, url := range info.FrequentURLs {
		fmt.Fprintf(out, "| `%s` | %d\n", url.Name, url.Quantity)
	}

	fmt.Fprint(out, "|===\n\n")

	fmt.Fprint(out, "==== Response Codes\n\n")
	fmt.Fprint(out, "[options=\"header\"]\n")
	fmt.Fprint(out, "|===\n")
	fmt.Fprint(out, "| Code | Name | Count\n")

	for _, status := range info.FrequentStatuses {
		fmt.Fprintf(out, "| %d | %s | %d\n", status.Code, status.Name, status.Quantity)
	}

	fmt.Fprint(out, "|===\n\n")

	fmt.Fprint(out, "==== Requesting addresses\n\n")
	fmt.Fprint(out, "[options=\"header\"]\n")
	fmt.Fprint(out, "|===\n")
	fmt.Fprint(out, "| Name | Count\n")

	for _, address := range info.FrequentAddresses {
		fmt.Fprintf(out, "| %s | %d\n", address.Name, address.Quantity)
	}

	fmt.Fprint(out, "|===\n")
}
