package parser_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/es-debug/backend-academy-2024-go-template/internal/domain"
	"github.com/es-debug/backend-academy-2024-go-template/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFile(t *testing.T) {
	tt := []struct {
		name             string
		content          string
		totalRequests    int
		avgResponseSize  int
		responseSize95p  int
		frequentURLs     []domain.URL
		frequentStatuses []domain.Status
	}{
		{
			name: "one log line",
			content: `130.41.23.21 - - [22/Oct/2024:09:48:45 +0000] ` +
				`"GET /clear-thinking%20Streamlined/architecture/background%20analyzing.gif ` +
				`HTTP/1.1" 200 2232 "-" ` +
				`"Opera/10.89 (Windows 98; Win 9x 4.90; en-US) ` +
				`Presto/2.13.253 Version/12.00"`,
			totalRequests:   1,
			avgResponseSize: 2232,
			responseSize95p: 2232,
			frequentURLs: []domain.URL{
				domain.NewURL(
					"/clear-thinking%20Streamlined/architecture/background%20analyzing.gif",
					1,
				),
			},
			frequentStatuses: []domain.Status{domain.NewStatus(http.StatusOK, http.StatusText(http.StatusOK), 1)},
		},
		{
			name: "multiple log lines",
			content: `33.114.0.221 - - [22/Oct/2024:09:48:45 +0000] "HEAD /Digitized/open%20system_hierarchy/moratorium.php ` +
				`HTTP/1.1" 200 2418 "-" ` +
				`"Mozilla/5.0 (Macintosh; U; PPC Mac OS X 10_8_7) AppleWebKit/5332 ` +
				`(KHTML, like Gecko) Chrome/37.0.891.0 Mobile Safari/5332"` + "\n" +

				`142.254.109.16 - - [22/Oct/2024:09:48:45 +0000] "GET /reciprocal/complexity.css ` +
				`HTTP/1.1" 200 2668 "-" ` +
				`"Mozilla/5.0 (Windows 95) AppleWebKit/5322 ` +
				`(KHTML, like Gecko) Chrome/37.0.824.0 Mobile Safari/5322"` + "\n" +

				`46.83.49.199 - - [22/Oct/2024:09:48:45 +0000] "GET /Triple-buffered.jpg ` +
				`HTTP/1.1" 200 922 "-" ` +
				`"Mozilla/5.0 (X11; Linux i686; rv:8.0) Gecko/2024-12-09 Firefox/35.0"` + "\n" +

				`8.177.148.191 - - [22/Oct/2024:09:48:45 +0000] "HEAD /Triple-buffered.jpg ` +
				`HTTP/1.1" 404 92 "-" ` +
				`"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_0) AppleWebKit/5330 ` +
				`(KHTML, like Gecko) Chrome/38.0.825.0 Mobile Safari/5330"` + "\n" +

				`192.93.214.163 - - [22/Oct/2024:09:48:45 +0000] "POST /reciprocal/complexity.css ` +
				`HTTP/1.1" 300 2814 "-" ` +
				`"Mozilla/5.0 (X11; Linux i686) AppleWebKit/5330 ` +
				`(KHTML, like Gecko) Chrome/37.0.829.0 Mobile Safari/5330"`,
			totalRequests:   5,
			avgResponseSize: 1782,
			responseSize95p: 2814,
			frequentURLs: []domain.URL{
				domain.NewURL("/Triple-buffered.jpg", 2),
				domain.NewURL("/reciprocal/complexity.css", 2),
				domain.NewURL("/Digitized/open%20system_hierarchy/moratorium.php", 1),
			},
			frequentStatuses: []domain.Status{
				domain.NewStatus(http.StatusOK, http.StatusText(http.StatusOK), 3),
				domain.NewStatus(http.StatusMultipleChoices, http.StatusText(http.StatusMultipleChoices), 1),
				domain.NewStatus(http.StatusNotFound, http.StatusText(http.StatusNotFound), 1),
			},
		},
		{
			name:             "no log lines",
			content:          ``,
			totalRequests:    0,
			avgResponseSize:  0,
			responseSize95p:  0,
			frequentURLs:     nil,
			frequentStatuses: nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			fileName := createTestFiles(t, tc.content)
			defer deleteTestFiles(t, getRoot(fileName))

			logParser := parser.NewParser()

			data, err := logParser.Parse(fileName, nil, nil)
			require.NoError(t, err, "file must be parsed")

			assert.Equal(t, tc.totalRequests, data.TotalRequests)
			assert.Equal(t, tc.avgResponseSize, data.AvgResponseSize)
			assert.Equal(t, tc.responseSize95p, data.ResponseSize95p)
			assert.Equal(t, tc.frequentURLs, data.FrequentURLs)
			assert.Equal(t, tc.frequentStatuses, data.FrequentStatuses)
		})
	}
}

func TestParseMultipleFiles(t *testing.T) {
	tt := []struct {
		name             string
		content          []string
		totalRequests    int
		avgResponseSize  int
		responseSize95p  int
		frequentURLs     []domain.URL
		frequentStatuses []domain.Status
	}{
		{
			name: "multiple log lines",
			content: []string{
				`33.114.0.221 - - [22/Oct/2024:09:48:45 +0000] "HEAD /Digitized/open%20system_hierarchy/moratorium.php ` +
					`HTTP/1.1" 200 2418 "-" ` +
					`"Mozilla/5.0 (Macintosh; U; PPC Mac OS X 10_8_7) AppleWebKit/5332 ` +
					`(KHTML, like Gecko) Chrome/37.0.891.0 Mobile Safari/5332"`,

				`142.254.109.16 - - [22/Oct/2024:09:48:45 +0000] "GET /reciprocal/complexity.css ` +
					`HTTP/1.1" 200 2668 "-" ` +
					`"Mozilla/5.0 (Windows 95) AppleWebKit/5322 ` +
					`(KHTML, like Gecko) Chrome/37.0.824.0 Mobile Safari/5322"`,

				`46.83.49.199 - - [22/Oct/2024:09:48:45 +0000] "GET /Triple-buffered.jpg ` +
					`HTTP/1.1" 200 922 "-" ` +
					`"Mozilla/5.0 (X11; Linux i686; rv:8.0) Gecko/2024-12-09 Firefox/35.0"`,

				`8.177.148.191 - - [22/Oct/2024:09:48:45 +0000] "HEAD /Triple-buffered.jpg ` +
					`HTTP/1.1" 404 92 "-" ` +
					`"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_0) AppleWebKit/5330 ` +
					`(KHTML, like Gecko) Chrome/38.0.825.0 Mobile Safari/5330"`,

				`192.93.214.163 - - [22/Oct/2024:09:48:45 +0000] "POST /reciprocal/complexity.css ` +
					`HTTP/1.1" 300 2814 "-" ` +
					`"Mozilla/5.0 (X11; Linux i686) AppleWebKit/5330 ` +
					`(KHTML, like Gecko) Chrome/37.0.829.0 Mobile Safari/5330"`,
			},
			totalRequests:   5,
			avgResponseSize: 1782,
			responseSize95p: 2814,
			frequentURLs: []domain.URL{
				domain.NewURL("/Triple-buffered.jpg", 2),
				domain.NewURL("/reciprocal/complexity.css", 2),
				domain.NewURL("/Digitized/open%20system_hierarchy/moratorium.php", 1),
			},
			frequentStatuses: []domain.Status{
				domain.NewStatus(http.StatusOK, http.StatusText(http.StatusOK), 3),
				domain.NewStatus(http.StatusMultipleChoices, http.StatusText(http.StatusMultipleChoices), 1),
				domain.NewStatus(http.StatusNotFound, http.StatusText(http.StatusNotFound), 1),
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			fileName := createTestFiles(t, tc.content...)
			defer deleteTestFiles(t, getRoot(fileName))

			logParser := parser.NewParser()

			data, err := logParser.Parse(fileName, nil, nil)
			require.NoError(t, err, "file must be parsed")

			assert.Equal(t, tc.totalRequests, data.TotalRequests)
			assert.Equal(t, tc.avgResponseSize, data.AvgResponseSize)
			assert.Equal(t, tc.responseSize95p, data.ResponseSize95p)
			assert.Equal(t, tc.frequentURLs, data.FrequentURLs)
			assert.Equal(t, tc.frequentStatuses, data.FrequentStatuses)
		})
	}
}

func TestParseFileWithTimeFilter(t *testing.T) {
	tt := []struct {
		name             string
		content          string
		from             *time.Time
		to               *time.Time
		totalRequests    int
		avgResponseSize  int
		responseSize95p  int
		frequentURLs     []domain.URL
		frequentStatuses []domain.Status
	}{
		{
			name: "only from time",
			content: `130.41.23.21 - - [22/Oct/2024:09:48:45 +0000] ` +
				`"GET /clear-thinking%20Streamlined/architecture/background%20analyzing.gif ` +
				`HTTP/1.1" 200 2232 "-" ` +
				`"Opera/10.89 (Windows 98; Win 9x 4.90; en-US) ` +
				`Presto/2.13.253 Version/12.00"`,
			from:             getTime(t, "23/Oct/2024"),
			to:               nil,
			totalRequests:    0,
			avgResponseSize:  0,
			responseSize95p:  0,
			frequentURLs:     nil,
			frequentStatuses: nil,
		},
		{
			name: "only to time",
			content: `6.60.120.55 - - [23/Oct/2024:09:48:45 +0000] "HEAD /client-server-architecture.htm ` +
				`HTTP/1.1" 200 1286 "-" ` +
				`"Mozilla/5.0 (Windows; U; Windows NT 5.0) AppleWebKit/534.40.6 ` +
				`(KHTML, like Gecko) Version/4.0 Safari/534.40.6"` + "\n" +

				`174.118.205.41 - - [24/Oct/2024:09:48:45 +0000] "POST /Synchronised/mission-critical.jpg ` +
				`HTTP/1.1" 200 2739 "-" "Mozilla/5.0 (Windows NT 4.0) AppleWebKit/5330 ` +
				`(KHTML, like Gecko) Chrome/40.0.836.0 Mobile Safari/5330"` + "\n" +

				`5.69.24.249 - - [25/Oct/2024:09:48:45 +0000] "POST /Expanded-3rd%20generation-Ergonomic/bandwidth-monitored-pricing%20structure.php ` +
				`HTTP/1.1" 200 1354 "-" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/5341 ` +
				`(KHTML, like Gecko) Chrome/39.0.819.0 Mobile Safari/5341"` + "\n" +

				`172.227.236.171 - - [26/Oct/2024:09:48:45 +0000] "GET /time-frame%20secondary/encryption/secondary.php ` +
				`HTTP/1.1" 200 2415 "-" "Mozilla/5.0 (X11; Linux i686; rv:7.0) ` +
				`Gecko/1946-01-05 Firefox/36.0"` + "\n" +

				`124.254.231.79 - - [27/Oct/2024:09:48:45 +0000] "PUT /Customizable/complexity%20matrix-Graphical%20User%20Interface.svg ` +
				`HTTP/1.1" 200 1844 "-" "Mozilla/5.0 (Macintosh; PPC Mac OS X 10_7_7 rv:6.0; en-US) AppleWebKit/533.23.2 ` +
				`(KHTML, like Gecko) Version/4.2 Safari/533.23.2"`,
			from:            nil,
			to:              getTime(t, "24/Oct/2024"),
			totalRequests:   2,
			avgResponseSize: 2012,
			responseSize95p: 2739,
			frequentURLs: []domain.URL{
				domain.NewURL("/Synchronised/mission-critical.jpg", 1),
				domain.NewURL("/client-server-architecture.htm", 1),
			},
			frequentStatuses: []domain.Status{
				domain.NewStatus(http.StatusOK, http.StatusText(http.StatusOK), 2),
			},
		},
		{
			name: "from and to times",
			content: `33.114.0.221 - - [22/Oct/2024:09:48:45 +0000] "HEAD /Digitized/open%20system_hierarchy/moratorium.php ` +
				`HTTP/1.1" 200 2418 "-" ` +
				`"Mozilla/5.0 (Macintosh; U; PPC Mac OS X 10_8_7) AppleWebKit/5332 ` +
				`(KHTML, like Gecko) Chrome/37.0.891.0 Mobile Safari/5332"` + "\n" +

				`142.254.109.16 - - [23/Oct/2024:09:48:45 +0000] "GET /reciprocal/complexity.css ` +
				`HTTP/1.1" 200 2668 "-" ` +
				`"Mozilla/5.0 (Windows 95) AppleWebKit/5322 ` +
				`(KHTML, like Gecko) Chrome/37.0.824.0 Mobile Safari/5322"` + "\n" +

				`46.83.49.199 - - [24/Oct/2024:09:48:45 +0000] "GET /Triple-buffered.jpg ` +
				`HTTP/1.1" 200 922 "-" ` +
				`"Mozilla/5.0 (X11; Linux i686; rv:8.0) Gecko/2024-12-09 Firefox/35.0"` + "\n" +

				`8.177.148.191 - - [25/Oct/2024:09:48:45 +0000] "HEAD /Triple-buffered.jpg ` +
				`HTTP/1.1" 404 92 "-" ` +
				`"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_0) AppleWebKit/5330 ` +
				`(KHTML, like Gecko) Chrome/38.0.825.0 Mobile Safari/5330"` + "\n" +

				`192.93.214.163 - - [26/Oct/2024:09:48:45 +0000] "POST /reciprocal/complexity.css ` +
				`HTTP/1.1" 300 2814 "-" ` +
				`"Mozilla/5.0 (X11; Linux i686) AppleWebKit/5330 ` +
				`(KHTML, like Gecko) Chrome/37.0.829.0 Mobile Safari/5330"`,
			from:            getTime(t, "23/Oct/2024"),
			to:              getTime(t, "25/Oct/2024"),
			totalRequests:   3,
			avgResponseSize: 1227,
			responseSize95p: 2668,
			frequentURLs: []domain.URL{
				domain.NewURL("/Triple-buffered.jpg", 2),
				domain.NewURL("/reciprocal/complexity.css", 1),
			},
			frequentStatuses: []domain.Status{
				domain.NewStatus(http.StatusOK, http.StatusText(http.StatusOK), 2),
				domain.NewStatus(http.StatusNotFound, http.StatusText(http.StatusNotFound), 1),
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			fileName := createTestFiles(t, tc.content)
			defer deleteTestFiles(t, getRoot(fileName))

			logParser := parser.NewParser()

			data, err := logParser.Parse(fileName, tc.from, tc.to)
			require.NoError(t, err, "file must be parsed")

			assert.Equal(t, tc.totalRequests, data.TotalRequests)
			assert.Equal(t, tc.avgResponseSize, data.AvgResponseSize)
			assert.Equal(t, tc.responseSize95p, data.ResponseSize95p)
			assert.Equal(t, tc.frequentURLs, data.FrequentURLs)
			assert.Equal(t, tc.frequentStatuses, data.FrequentStatuses)
		})
	}
}

func TestParseFileContentError(t *testing.T) {
	tt := []struct {
		name    string
		content string
	}{
		{
			name:    "bad content",
			content: "bad content",
		},
		{
			name: "bad time",
			content: `219.251.118.203 - - [bad time] "GET /methodology/systemic_Phased-user-facing.php ` +
				`HTTP/1.1" 200 1040 "-" ` +
				`"Mozilla/5.0 (iPhone; CPU iPhone OS 8_0_2 like Mac OS X; en-US) ` +
				`AppleWebKit/532.6.6 (KHTML, like Gecko) Version/4.0.5 ` +
				`Mobile/8B117 Safari/6532.6.6"`,
		},
		{
			name: "no such status",
			content: `219.251.118.203 - - [22/Oct/2024:09:48:45 +0000] "GET /methodology/systemic_Phased-user-facing.php ` +
				`HTTP/1.1" 199 1040 "-" ` +
				`"Mozilla/5.0 (iPhone; CPU iPhone OS 8_0_2 like Mac OS X; en-US) ` +
				`AppleWebKit/532.6.6 (KHTML, like Gecko) Version/4.0.5 ` +
				`Mobile/8B117 Safari/6532.6.6"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			fileName := createTestFiles(t, tc.content)
			defer deleteTestFiles(t, getRoot(fileName))

			logParser := parser.NewParser()

			_, err := logParser.Parse(fileName, nil, nil)
			require.Error(t, err, "bad content")
		})
	}
}

func TestParseFileExistenceError(t *testing.T) {
	tt := []struct {
		name     string
		fileName string
	}{
		{
			name:     "no such file",
			fileName: fmt.Sprintf("bad file name %d", time.Now().UnixNano()),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logParser := parser.NewParser()

			_, err := logParser.Parse(tc.fileName, nil, nil)
			require.Error(t, err, "bad content")
		})
	}
}

func TestParseURL(t *testing.T) {
	tt := []struct {
		name             string
		content          string
		totalRequests    int
		avgResponseSize  int
		responseSize95p  int
		frequentURLs     []domain.URL
		frequentStatuses []domain.Status
	}{
		{
			name: "multiple log lines",
			content: `219.251.118.203 - - [22/Oct/2024:09:48:45 +0000] "GET /methodology/systemic_Phased-user-facing.php ` +
				`HTTP/1.1" 200 1040 "-" ` +
				`"Mozilla/5.0 (iPhone; CPU iPhone OS 8_0_2 like Mac OS X; en-US) ` +
				`AppleWebKit/532.6.6 (KHTML, like Gecko) Version/4.0.5 ` +
				`Mobile/8B117 Safari/6532.6.6"` + "\n" +
				`45.175.78.55 - - [22/Oct/2024:09:48:45 +0000] "GET /actuating_5th%20generation-Multi-channelled/` +
				`application/Multi-lateral.png HTTP/1.1" 200 1953 "-" ` +
				`"Mozilla/5.0 (Macintosh; U; Intel Mac OS X 10_7_0 rv:2.0) ` +
				`Gecko/1951-31-07 Firefox/36.0"`,

			totalRequests:   2,
			avgResponseSize: 1496,
			responseSize95p: 1953,
			frequentURLs: []domain.URL{
				domain.NewURL(
					"/actuating_5th%20generation-Multi-channelled/application/Multi-lateral.png",
					1,
				),
				domain.NewURL("/methodology/systemic_Phased-user-facing.php", 1),
			},
			frequentStatuses: []domain.Status{domain.NewStatus(200, http.StatusText(200), 2)},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					fmt.Fprint(w, tc.content)
				}),
			)
			defer server.Close()

			logParser := parser.NewParser()

			data, err := logParser.Parse(server.URL, nil, nil)
			require.NoError(t, err, "must parse data from server")

			assert.Equal(t, tc.totalRequests, data.TotalRequests)
			assert.Equal(t, tc.avgResponseSize, data.AvgResponseSize)
			assert.Equal(t, tc.responseSize95p, data.ResponseSize95p)
			assert.Equal(t, tc.frequentURLs, data.FrequentURLs)
			assert.Equal(t, tc.frequentStatuses, data.FrequentStatuses)
		})
	}
}

func TestParseURLError(t *testing.T) {
	tt := []struct {
		name string
		url  string
	}{
		{
			name: "bad url",
			url:  "bad url",
		},
		{
			name: `bad url starts with "http://"`,
			url:  "http://bad_url",
		},
	}

	for i, tc := range tt {
		t.Run(fmt.Sprintf("#%d", i+1), func(t *testing.T) {
			logParser := parser.NewParser()

			_, err := logParser.Parse(tc.url, nil, nil)
			require.Error(t, err, "bad url")
		})
	}
}
