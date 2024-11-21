package parser_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LLIEPJIOK/nginxparser/internal/domain"
	"github.com/LLIEPJIOK/nginxparser/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFile(t *testing.T) {
	tt := []struct {
		name              string
		content           string
		totalRequests     int
		avgResponseSize   int
		responseSize95p   int
		avgRequestsPerDay int
		frequentURLs      []domain.URL
		frequentStatuses  []domain.Status
		frequentAddresses []domain.Address
	}{
		{
			name: "one log line",
			content: `130.41.23.21 - - [22/Oct/2024:09:48:45 +0000] ` +
				`"GET /clear-thinking%20Streamlined/architecture/background%20analyzing.gif ` +
				`HTTP/1.1" 200 2232 "-" ` +
				`"Opera/10.89 (Windows 98; Win 9x 4.90; en-US) ` +
				`Presto/2.13.253 Version/12.00"`,
			totalRequests:     1,
			avgResponseSize:   2232,
			responseSize95p:   2232,
			avgRequestsPerDay: 1,
			frequentURLs: []domain.URL{
				domain.NewURL(
					"/clear-thinking%20Streamlined/architecture/background%20analyzing.gif",
					1,
				),
			},
			frequentStatuses: []domain.Status{
				domain.NewStatus(200, 1),
			},
			frequentAddresses: []domain.Address{domain.NewAddress("130.41.23.21", 1)},
		},
		{
			name: "multiple log lines",
			content: `33.114.0.221 - - [22/Oct/2024:09:48:45 +0000] "HEAD /Digitized/open%20system_hierarchy/moratorium.php ` +
				`HTTP/1.1" 200 2418 "-" ` +
				`"Mozilla/5.0 (Macintosh; U; PPC Mac OS X 10_8_7) AppleWebKit/5332 ` +
				`(KHTML, like Gecko) Chrome/37.0.891.0 Mobile Safari/5332"` + "\n" +

				`192.93.214.163 - - [22/Oct/2024:09:48:45 +0000] "GET /reciprocal/complexity.css ` +
				`HTTP/1.1" 200 2668 "-" ` +
				`"Mozilla/5.0 (Windows 95) AppleWebKit/5322 ` +
				`(KHTML, like Gecko) Chrome/37.0.824.0 Mobile Safari/5322"` + "\n" +

				`192.93.214.163 - - [22/Oct/2024:09:48:45 +0000] "GET /Triple-buffered.jpg ` +
				`HTTP/1.1" 200 922 "-" ` +
				`"Mozilla/5.0 (X11; Linux i686; rv:8.0) Gecko/2024-12-09 Firefox/35.0"` + "\n" +

				`192.93.214.163 - - [23/Oct/2024:09:48:45 +0000] "HEAD /Triple-buffered.jpg ` +
				`HTTP/1.1" 404 92 "-" ` +
				`"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_0) AppleWebKit/5330 ` +
				`(KHTML, like Gecko) Chrome/38.0.825.0 Mobile Safari/5330"` + "\n" +

				`192.93.214.163 - - [23/Oct/2024:09:48:45 +0000] "POST /reciprocal/complexity.css ` +
				`HTTP/1.1" 300 2814 "-" ` +
				`"Mozilla/5.0 (X11; Linux i686) AppleWebKit/5330 ` +
				`(KHTML, like Gecko) Chrome/37.0.829.0 Mobile Safari/5330"`,
			totalRequests:     5,
			avgResponseSize:   1782,
			responseSize95p:   2814,
			avgRequestsPerDay: 2,
			frequentURLs: []domain.URL{
				domain.NewURL("/Triple-buffered.jpg", 2),
				domain.NewURL("/reciprocal/complexity.css", 2),
				domain.NewURL("/Digitized/open%20system_hierarchy/moratorium.php", 1),
			},
			frequentStatuses: []domain.Status{
				domain.NewStatus(200, 3),
				domain.NewStatus(
					300,
					1,
				),
				domain.NewStatus(404, 1),
			},
			frequentAddresses: []domain.Address{
				domain.NewAddress("192.93.214.163", 4),
				domain.NewAddress("33.114.0.221", 1),
			},
		},
		{
			name:              "no log lines",
			content:           ``,
			totalRequests:     0,
			avgResponseSize:   0,
			responseSize95p:   0,
			avgRequestsPerDay: 0,
			frequentURLs:      nil,
			frequentStatuses:  nil,
			frequentAddresses: nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			fileName := createTestFiles(t, tc.content)
			defer deleteTestFiles(t, getRoot(fileName))

			logParser := parser.New()

			data, err := logParser.Parse(parser.Params{
				Path: fileName,
			})
			require.NoError(t, err, "file must be parsed")

			assert.Equal(t, tc.totalRequests, data.TotalRequests)
			assert.Equal(t, tc.avgResponseSize, data.AvgResponseSize)
			assert.Equal(t, tc.responseSize95p, data.ResponseSize95p)
			assert.Equal(t, tc.avgRequestsPerDay, data.AvgResponsePerDay)
			assert.Equal(t, tc.frequentURLs, data.FrequentURLs)
			assert.Equal(t, tc.frequentStatuses, data.FrequentStatuses)
			assert.Equal(t, tc.frequentAddresses, data.FrequentAddresses)
		})
	}
}

func TestParseMultipleFiles(t *testing.T) {
	tt := []struct {
		name              string
		content           []string
		totalRequests     int
		avgResponseSize   int
		responseSize95p   int
		avgRequestsPerDay int
		frequentURLs      []domain.URL
		frequentStatuses  []domain.Status
		frequentAddresses []domain.Address
	}{
		{
			name: "multiple log lines",
			content: []string{
				`192.93.214.163 - - [22/Oct/2024:09:48:45 +0000] "HEAD /Digitized/open%20system_hierarchy/moratorium.php ` +
					`HTTP/1.1" 200 2418 "-" ` +
					`"Mozilla/5.0 (Macintosh; U; PPC Mac OS X 10_8_7) AppleWebKit/5332 ` +
					`(KHTML, like Gecko) Chrome/37.0.891.0 Mobile Safari/5332"`,

				`46.83.49.199 - - [23/Oct/2024:09:48:45 +0000] "GET /reciprocal/complexity.css ` +
					`HTTP/1.1" 200 2668 "-" ` +
					`"Mozilla/5.0 (Windows 95) AppleWebKit/5322 ` +
					`(KHTML, like Gecko) Chrome/37.0.824.0 Mobile Safari/5322"`,

				`46.83.49.199 - - [22/Oct/2024:09:48:45 +0000] "GET /Triple-buffered.jpg ` +
					`HTTP/1.1" 200 922 "-" ` +
					`"Mozilla/5.0 (X11; Linux i686; rv:8.0) Gecko/2024-12-09 Firefox/35.0"`,

				`8.177.148.191 - - [24/Oct/2024:09:48:45 +0000] "HEAD /Triple-buffered.jpg ` +
					`HTTP/1.1" 404 92 "-" ` +
					`"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_0) AppleWebKit/5330 ` +
					`(KHTML, like Gecko) Chrome/38.0.825.0 Mobile Safari/5330"`,

				`192.93.214.163 - - [25/Oct/2024:09:48:45 +0000] "POST /reciprocal/complexity.css ` +
					`HTTP/1.1" 300 2814 "-" ` +
					`"Mozilla/5.0 (X11; Linux i686) AppleWebKit/5330 ` +
					`(KHTML, like Gecko) Chrome/37.0.829.0 Mobile Safari/5330"`,
			},
			totalRequests:     5,
			avgResponseSize:   1782,
			responseSize95p:   2814,
			avgRequestsPerDay: 1,
			frequentURLs: []domain.URL{
				domain.NewURL("/Triple-buffered.jpg", 2),
				domain.NewURL("/reciprocal/complexity.css", 2),
				domain.NewURL("/Digitized/open%20system_hierarchy/moratorium.php", 1),
			},
			frequentStatuses: []domain.Status{
				domain.NewStatus(200, 3),
				domain.NewStatus(
					300,
					1,
				),
				domain.NewStatus(404, 1),
			},
			frequentAddresses: []domain.Address{
				domain.NewAddress("192.93.214.163", 2),
				domain.NewAddress("46.83.49.199", 2),
				domain.NewAddress("8.177.148.191", 1),
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			fileName := createTestFiles(t, tc.content...)
			defer deleteTestFiles(t, getRoot(fileName))

			logParser := parser.New()

			data, err := logParser.Parse(parser.Params{
				Path: fileName,
			})
			require.NoError(t, err, "file must be parsed")

			assert.Equal(t, tc.totalRequests, data.TotalRequests)
			assert.Equal(t, tc.avgResponseSize, data.AvgResponseSize)
			assert.Equal(t, tc.responseSize95p, data.ResponseSize95p)
			assert.Equal(t, tc.avgRequestsPerDay, data.AvgResponsePerDay)
			assert.Equal(t, tc.frequentURLs, data.FrequentURLs)
			assert.Equal(t, tc.frequentStatuses, data.FrequentStatuses)
			assert.Equal(t, tc.frequentAddresses, data.FrequentAddresses)
		})
	}
}

func TestParseFileWithTimeFilter(t *testing.T) {
	tt := []struct {
		name              string
		content           string
		from              *time.Time
		to                *time.Time
		totalRequests     int
		avgResponseSize   int
		avgRequestsPerDay int
		responseSize95p   int
		frequentURLs      []domain.URL
		frequentStatuses  []domain.Status
		frequentAddresses []domain.Address
	}{
		{
			name: "only from time",
			content: `130.41.23.21 - - [22/Oct/2024:09:48:45 +0000] ` +
				`"GET /clear-thinking%20Streamlined/architecture/background%20analyzing.gif ` +
				`HTTP/1.1" 200 2232 "-" ` +
				`"Opera/10.89 (Windows 98; Win 9x 4.90; en-US) ` +
				`Presto/2.13.253 Version/12.00"`,
			from:              getTime(t, "23/Oct/2024"),
			to:                nil,
			totalRequests:     0,
			avgResponseSize:   0,
			responseSize95p:   0,
			avgRequestsPerDay: 0,
			frequentURLs:      nil,
			frequentStatuses:  nil,
			frequentAddresses: nil,
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
			from:              nil,
			to:                getTime(t, "24/Oct/2024"),
			totalRequests:     2,
			avgResponseSize:   2012,
			responseSize95p:   2739,
			avgRequestsPerDay: 1,
			frequentURLs: []domain.URL{
				domain.NewURL("/Synchronised/mission-critical.jpg", 1),
				domain.NewURL("/client-server-architecture.htm", 1),
			},
			frequentStatuses: []domain.Status{
				domain.NewStatus(200, 2),
			},
			frequentAddresses: []domain.Address{
				domain.NewAddress("174.118.205.41", 1),
				domain.NewAddress("6.60.120.55", 1),
			},
		},
		{
			name: "from and to times",
			content: `33.114.0.221 - - [22/Oct/2024:09:48:45 +0000] "HEAD /Digitized/open%20system_hierarchy/moratorium.php ` +
				`HTTP/1.1" 200 2418 "-" ` +
				`"Mozilla/5.0 (Macintosh; U; PPC Mac OS X 10_8_7) AppleWebKit/5332 ` +
				`(KHTML, like Gecko) Chrome/37.0.891.0 Mobile Safari/5332"` + "\n" +

				`8.177.148.191 - - [23/Oct/2024:09:48:45 +0000] "GET /reciprocal/complexity.css ` +
				`HTTP/1.1" 200 2668 "-" ` +
				`"Mozilla/5.0 (Windows 95) AppleWebKit/5322 ` +
				`(KHTML, like Gecko) Chrome/37.0.824.0 Mobile Safari/5322"` + "\n" +

				`8.177.148.191 - - [24/Oct/2024:09:48:45 +0000] "GET /Triple-buffered.jpg ` +
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
			from:              getTime(t, "23/Oct/2024"),
			to:                getTime(t, "25/Oct/2024"),
			totalRequests:     3,
			avgResponseSize:   1227,
			responseSize95p:   2668,
			avgRequestsPerDay: 1,
			frequentURLs: []domain.URL{
				domain.NewURL("/Triple-buffered.jpg", 2),
				domain.NewURL("/reciprocal/complexity.css", 1),
			},
			frequentStatuses: []domain.Status{
				domain.NewStatus(200, 2),
				domain.NewStatus(404, 1),
			},
			frequentAddresses: []domain.Address{domain.NewAddress("8.177.148.191", 3)},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			fileName := createTestFiles(t, tc.content)
			defer deleteTestFiles(t, getRoot(fileName))

			logParser := parser.New()

			data, err := logParser.Parse(parser.Params{
				Path: fileName,
				From: tc.from,
				To:   tc.to,
			})
			require.NoError(t, err, "file must be parsed")

			assert.Equal(t, tc.totalRequests, data.TotalRequests)
			assert.Equal(t, tc.avgResponseSize, data.AvgResponseSize)
			assert.Equal(t, tc.responseSize95p, data.ResponseSize95p)
			assert.Equal(t, tc.avgRequestsPerDay, data.AvgResponsePerDay)
			assert.Equal(t, tc.frequentURLs, data.FrequentURLs)
			assert.Equal(t, tc.frequentStatuses, data.FrequentStatuses)
			assert.Equal(t, tc.frequentAddresses, data.FrequentAddresses)
		})
	}
}

func TestParseFileWithFieldFilter(t *testing.T) {
	tt := []struct {
		name              string
		content           string
		field             string
		value             string
		totalRequests     int
		avgResponseSize   int
		avgRequestsPerDay int
		responseSize95p   int
		frequentURLs      []domain.URL
		frequentStatuses  []domain.Status
		frequentAddresses []domain.Address
	}{
		{
			name: "only field",
			content: `130.41.23.21 - - [22/Oct/2024:09:48:45 +0000] ` +
				`"GET /clear-thinking%20Streamlined/architecture/background%20analyzing.gif ` +
				`HTTP/1.1" 200 2232 "-" ` +
				`"Opera/10.89 (Windows 98; Win 9x 4.90; en-US) ` +
				`Presto/2.13.253 Version/12.00"`,
			field:             "method",
			value:             "",
			totalRequests:     1,
			avgResponseSize:   2232,
			responseSize95p:   2232,
			avgRequestsPerDay: 1,
			frequentURLs: []domain.URL{
				domain.NewURL(
					"/clear-thinking%20Streamlined/architecture/background%20analyzing.gif",
					1,
				),
			},
			frequentStatuses:  []domain.Status{domain.NewStatus(200, 1)},
			frequentAddresses: []domain.Address{domain.NewAddress("130.41.23.21", 1)},
		},
		{
			name: "only value",
			content: `130.41.23.21 - - [22/Oct/2024:09:48:45 +0000] ` +
				`"GET /clear-thinking%20Streamlined/architecture/background%20analyzing.gif ` +
				`HTTP/1.1" 200 2232 "-" ` +
				`"Opera/10.89 (Windows 98; Win 9x 4.90; en-US) ` +
				`Presto/2.13.253 Version/12.00"`,
			field:             "",
			value:             "value",
			totalRequests:     1,
			avgResponseSize:   2232,
			responseSize95p:   2232,
			avgRequestsPerDay: 1,
			frequentURLs: []domain.URL{
				domain.NewURL(
					"/clear-thinking%20Streamlined/architecture/background%20analyzing.gif",
					1,
				),
			},
			frequentStatuses:  []domain.Status{domain.NewStatus(200, 1)},
			frequentAddresses: []domain.Address{domain.NewAddress("130.41.23.21", 1)},
		},
		{
			name: "Method",
			content: `130.41.23.21 - - [22/Oct/2024:09:48:45 +0000] ` +
				`"GET /clear-thinking%20Streamlined/architecture/background%20analyzing.gif ` +
				`HTTP/1.1" 200 2232 "-" ` +
				`"Opera/10.89 (Windows 98; Win 9x 4.90; en-US) ` +
				`Presto/2.13.253 Version/12.00"`,
			field:             "Method",
			value:             "POST",
			totalRequests:     0,
			avgResponseSize:   0,
			responseSize95p:   0,
			avgRequestsPerDay: 0,
			frequentURLs:      nil,
			frequentStatuses:  nil,
			frequentAddresses: nil,
		},
		{
			name: "TimeLocal",
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
			field:             "TimeLocal",
			value:             "25/Oct/2024",
			totalRequests:     1,
			avgResponseSize:   1354,
			responseSize95p:   1354,
			avgRequestsPerDay: 1,
			frequentURLs: []domain.URL{
				domain.NewURL(
					"/Expanded-3rd%20generation-Ergonomic/bandwidth-monitored-pricing%20structure.php",
					1,
				),
			},
			frequentStatuses: []domain.Status{
				domain.NewStatus(200, 1),
			},
			frequentAddresses: []domain.Address{
				domain.NewAddress("5.69.24.249", 1),
			},
		},
		{
			name: "RemoteAddress",
			content: `33.114.0.221 - - [22/Oct/2024:09:48:45 +0000] "HEAD /Digitized/open%20system_hierarchy/moratorium.php ` +
				`HTTP/1.1" 200 2418 "-" ` +
				`"Mozilla/5.0 (Macintosh; U; PPC Mac OS X 10_8_7) AppleWebKit/5332 ` +
				`(KHTML, like Gecko) Chrome/37.0.891.0 Mobile Safari/5332"` + "\n" +

				`8.177.148.191 - - [23/Oct/2024:09:48:45 +0000] "GET /reciprocal/complexity.css ` +
				`HTTP/1.1" 200 2668 "-" ` +
				`"Mozilla/5.0 (Windows 95) AppleWebKit/5322 ` +
				`(KHTML, like Gecko) Chrome/37.0.824.0 Mobile Safari/5322"` + "\n" +

				`8.177.148.191 - - [24/Oct/2024:09:48:45 +0000] "GET /Triple-buffered.jpg ` +
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
			field:             "RemoteAddress",
			value:             "^[83].*",
			totalRequests:     4,
			avgResponseSize:   1525,
			responseSize95p:   2668,
			avgRequestsPerDay: 1,
			frequentURLs: []domain.URL{
				domain.NewURL("/Triple-buffered.jpg", 2),
				domain.NewURL("/Digitized/open%20system_hierarchy/moratorium.php", 1),
				domain.NewURL("/reciprocal/complexity.css", 1),
			},
			frequentStatuses: []domain.Status{
				domain.NewStatus(200, 3),
				domain.NewStatus(404, 1),
			},
			frequentAddresses: []domain.Address{
				domain.NewAddress("8.177.148.191", 3),
				domain.NewAddress("33.114.0.221", 1),
			},
		},
		{
			name: "Status",
			content: `33.114.0.221 - - [22/Oct/2024:09:48:45 +0000] "HEAD /Digitized/open%20system_hierarchy/moratorium.php ` +
				`HTTP/1.1" 200 2418 "-" ` +
				`"Mozilla/5.0 (Macintosh; U; PPC Mac OS X 10_8_7) AppleWebKit/5332 ` +
				`(KHTML, like Gecko) Chrome/37.0.891.0 Mobile Safari/5332"` + "\n" +

				`8.177.148.191 - - [23/Oct/2024:09:48:45 +0000] "GET /reciprocal/complexity.css ` +
				`HTTP/1.1" 200 2668 "-" ` +
				`"Mozilla/5.0 (Windows 95) AppleWebKit/5322 ` +
				`(KHTML, like Gecko) Chrome/37.0.824.0 Mobile Safari/5322"` + "\n" +

				`8.177.148.191 - - [24/Oct/2024:09:48:45 +0000] "GET /Triple-buffered.jpg ` +
				`HTTP/1.1" 200 922 "-" ` +
				`"Mozilla/5.0 (X11; Linux i686; rv:8.0) Gecko/2024-12-09 Firefox/35.0"` + "\n" +

				`8.177.148.191 - - [25/Oct/2024:09:48:45 +0000] "HEAD /Triple-buffered.jpg ` +
				`HTTP/1.1" 404 92 "-" ` +
				`"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_0) AppleWebKit/5330 ` +
				`(KHTML, like Gecko) Chrome/38.0.825.0 Mobile Safari/5330"` + "\n" +

				`192.93.214.163 - - [25/Oct/2024:09:48:45 +0000] "POST /reciprocal/complexity.css ` +
				`HTTP/1.1" 304 2814 "-" ` +
				`"Mozilla/5.0 (X11; Linux i686) AppleWebKit/5330 ` +
				`(KHTML, like Gecko) Chrome/37.0.829.0 Mobile Safari/5330"`,
			field:             "Status",
			value:             ".04",
			totalRequests:     2,
			avgResponseSize:   1453,
			responseSize95p:   2814,
			avgRequestsPerDay: 2,
			frequentURLs: []domain.URL{
				domain.NewURL("/Triple-buffered.jpg", 1),
				domain.NewURL("/reciprocal/complexity.css", 1),
			},
			frequentStatuses: []domain.Status{
				domain.NewStatus(304, 1),
				domain.NewStatus(404, 1),
			},
			frequentAddresses: []domain.Address{
				domain.NewAddress("192.93.214.163", 1),
				domain.NewAddress("8.177.148.191", 1),
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			fileName := createTestFiles(t, tc.content)
			defer deleteTestFiles(t, getRoot(fileName))

			logParser := parser.New()

			data, err := logParser.Parse(parser.Params{
				Path:        fileName,
				FilterField: tc.field,
				FilterValue: tc.value,
			})
			require.NoError(t, err, "file must be parsed")

			assert.Equal(t, tc.totalRequests, data.TotalRequests)
			assert.Equal(t, tc.avgResponseSize, data.AvgResponseSize)
			assert.Equal(t, tc.responseSize95p, data.ResponseSize95p)
			assert.Equal(t, tc.avgRequestsPerDay, data.AvgResponsePerDay)
			assert.Equal(t, tc.frequentURLs, data.FrequentURLs)
			assert.Equal(t, tc.frequentStatuses, data.FrequentStatuses)
			assert.Equal(t, tc.frequentAddresses, data.FrequentAddresses)
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

			logParser := parser.New()

			_, err := logParser.Parse(parser.Params{
				Path: fileName,
			})
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
			logParser := parser.New()

			_, err := logParser.Parse(parser.Params{
				Path: tc.fileName,
			})
			require.Error(t, err, "bad content")
		})
	}
}

func TestParseURL(t *testing.T) {
	tt := []struct {
		name              string
		content           string
		totalRequests     int
		avgResponseSize   int
		responseSize95p   int
		avgRequestsPerDay int
		frequentURLs      []domain.URL
		frequentStatuses  []domain.Status
		frequentAddresses []domain.Address
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

			totalRequests:     2,
			avgResponseSize:   1496,
			responseSize95p:   1953,
			avgRequestsPerDay: 2,
			frequentURLs: []domain.URL{
				domain.NewURL(
					"/actuating_5th%20generation-Multi-channelled/application/Multi-lateral.png",
					1,
				),
				domain.NewURL("/methodology/systemic_Phased-user-facing.php", 1),
			},
			frequentStatuses: []domain.Status{domain.NewStatus(200, 2)},
			frequentAddresses: []domain.Address{
				domain.NewAddress("219.251.118.203", 1),
				domain.NewAddress("45.175.78.55", 1),
			},
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

			logParser := parser.New()

			data, err := logParser.Parse(parser.Params{
				Path: server.URL,
			})
			require.NoError(t, err, "must parse data from server")

			assert.Equal(t, tc.totalRequests, data.TotalRequests)
			assert.Equal(t, tc.avgResponseSize, data.AvgResponseSize)
			assert.Equal(t, tc.responseSize95p, data.ResponseSize95p)
			assert.Equal(t, tc.avgRequestsPerDay, data.AvgResponsePerDay)
			assert.Equal(t, tc.frequentURLs, data.FrequentURLs)
			assert.Equal(t, tc.frequentStatuses, data.FrequentStatuses)
			assert.Equal(t, tc.frequentAddresses, data.FrequentAddresses)
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
			logParser := parser.New()

			_, err := logParser.Parse(parser.Params{
				Path: tc.url,
			})
			require.Error(t, err, "bad url")
		})
	}
}

func TestMarkdown(t *testing.T) {
	tt := []struct {
		name     string
		info     *domain.FileInfo
		expected string
	}{
		{
			name: "Single file",
			info: &domain.FileInfo{
				Paths:             []string{"/var/log/nginx/access.log"},
				TotalRequests:     100,
				AvgResponseSize:   512,
				ResponseSize95p:   800,
				AvgResponsePerDay: 10,
				FrequentURLs: []domain.URL{
					domain.NewURL("/index.html", 50),
					domain.NewURL("/about.html", 20),
				},
				FrequentStatuses: []domain.Status{
					domain.NewStatus(200, 80),
					domain.NewStatus(404, 10),
				},
				FrequentAddresses: []domain.Address{
					domain.NewAddress("192.168.1.1", 30),
					domain.NewAddress("10.0.0.2", 20),
				},
			},
			expected: "#### General information\n\n" +
				"| Метрика | Значение |\n" +
				"|:-|-:|\n" +
				"| Files | /var/log/nginx/access.log |\n" +
				"| Number of requests | 100 |\n" +
				"| Average response size | 512 |\n" +
				"| 95th Percentile of response size | 800 |\n" +
				"| Average requests per day | 10 |\n\n" +
				"#### Requested resources\n\n" +
				"| Resource | Count |\n" +
				"|:-|-:|\n" +
				"| `/index.html` | 50 |\n" +
				"| `/about.html` | 20 |\n\n" +
				"#### Response codes\n\n" +
				"| Code | Name | Count |\n" +
				"|:-|:-:|-:|\n" +
				"| 200 | OK | 80 |\n" +
				"| 404 | Not Found | 10 |\n\n" +
				"#### Requesting addresses\n\n" +
				"| Address | Count |\n" +
				"|:-|-:|\n" +
				"| `192.168.1.1` | 30 |\n" +
				"| `10.0.0.2` | 20 |\n",
		},
		{
			name: "Multiple files",
			info: &domain.FileInfo{
				Paths:             []string{"/var/log/nginx/access.log", "/var/log/nginx/access.log.1"},
				TotalRequests:     1000,
				AvgResponseSize:   1024,
				ResponseSize95p:   1500,
				AvgResponsePerDay: 100,
				FrequentURLs: []domain.URL{
					domain.NewURL("/home", 300),
					domain.NewURL("/login", 150),
					domain.NewURL("/dashboard", 100),
				},
				FrequentStatuses: []domain.Status{
					domain.NewStatus(200, 700),
					domain.NewStatus(403, 50),
					domain.NewStatus(500, 20),
				},
				FrequentAddresses: []domain.Address{
					domain.NewAddress("172.16.0.1", 200),
					domain.NewAddress("192.168.1.2", 150),
					domain.NewAddress("10.0.0.3", 120),
				},
			},
			expected: "#### General information\n\n" +
				"| Метрика | Значение |\n" +
				"|:-|-:|\n" +
				"| Files | /var/log/nginx/access.log, /var/log/nginx/access.log.1 |\n" +
				"| Number of requests | 1000 |\n" +
				"| Average response size | 1024 |\n" +
				"| 95th Percentile of response size | 1500 |\n" +
				"| Average requests per day | 100 |\n\n" +
				"#### Requested resources\n\n" +
				"| Resource | Count |\n" +
				"|:-|-:|\n" +
				"| `/home` | 300 |\n" +
				"| `/login` | 150 |\n" +
				"| `/dashboard` | 100 |\n\n" +
				"#### Response codes\n\n" +
				"| Code | Name | Count |\n" +
				"|:-|:-:|-:|\n" +
				"| 200 | OK | 700 |\n" +
				"| 403 | Forbidden | 50 |\n" +
				"| 500 | Internal Server Error | 20 |\n\n" +
				"#### Requesting addresses\n\n" +
				"| Address | Count |\n" +
				"|:-|-:|\n" +
				"| `172.16.0.1` | 200 |\n" +
				"| `192.168.1.2` | 150 |\n" +
				"| `10.0.0.3` | 120 |\n",
		},
		{
			name: "URL",
			info: &domain.FileInfo{
				Paths:             []string{"https://raw.githubusercontent.com/elastic/examples/master/Common%20Data%20Formats/nginx_logs/nginx_logs"},
				TotalRequests:     5000,
				AvgResponseSize:   2048,
				ResponseSize95p:   3000,
				AvgResponsePerDay: 500,
				FrequentURLs: []domain.URL{
					domain.NewURL("/home", 1000),
					domain.NewURL("/products", 800),
					domain.NewURL("/contact", 600),
				},
				FrequentStatuses: []domain.Status{
					domain.NewStatus(200, 4000),
					domain.NewStatus(404, 400),
					domain.NewStatus(503, 50),
				},
				FrequentAddresses: []domain.Address{
					domain.NewAddress("192.168.0.10", 500),
					domain.NewAddress("192.168.0.20", 450),
					domain.NewAddress("192.168.0.30", 300),
				},
			},
			expected: "#### General information\n\n" +
				"| Метрика | Значение |\n" +
				"|:-|-:|\n" +
				"| Files | https://raw.githubusercontent.com/elastic/examples/master/Common%20Data%20Formats/nginx_logs/nginx_logs |\n" +
				"| Number of requests | 5000 |\n" +
				"| Average response size | 2048 |\n" +
				"| 95th Percentile of response size | 3000 |\n" +
				"| Average requests per day | 500 |\n\n" +
				"#### Requested resources\n\n" +
				"| Resource | Count |\n" +
				"|:-|-:|\n" +
				"| `/home` | 1000 |\n" +
				"| `/products` | 800 |\n" +
				"| `/contact` | 600 |\n\n" +
				"#### Response codes\n\n" +
				"| Code | Name | Count |\n" +
				"|:-|:-:|-:|\n" +
				"| 200 | OK | 4000 |\n" +
				"| 404 | Not Found | 400 |\n" +
				"| 503 | Service Unavailable | 50 |\n\n" +
				"#### Requesting addresses\n\n" +
				"| Address | Count |\n" +
				"|:-|-:|\n" +
				"| `192.168.0.10` | 500 |\n" +
				"| `192.168.0.20` | 450 |\n" +
				"| `192.168.0.30` | 300 |\n",
		},
	}

	logParser := parser.New()

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logParser.Markdown(tc.info, buf)

			assert.Equal(t, tc.expected, buf.String())
		})
	}
}

func TestAdoc(t *testing.T) {
	tt := []struct {
		name     string
		info     *domain.FileInfo
		expected string
	}{
		{
			name: "Single file",
			info: &domain.FileInfo{
				Paths:             []string{"/var/log/nginx/access.log"},
				TotalRequests:     100,
				AvgResponseSize:   512,
				ResponseSize95p:   800,
				AvgResponsePerDay: 10,
				FrequentURLs: []domain.URL{
					domain.NewURL("/index.html", 50),
					domain.NewURL("/about.html", 20),
				},
				FrequentStatuses: []domain.Status{
					domain.NewStatus(200, 80),
					domain.NewStatus(404, 10),
				},
				FrequentAddresses: []domain.Address{
					domain.NewAddress("192.168.1.1", 30),
					domain.NewAddress("10.0.0.2", 20),
				},
			},
			expected: "==== General Information\n\n" +
				"[options=\"header\"]\n" +
				"|===\n" +
				"| Метрика | Значение\n" +
				"| Files | /var/log/nginx/access.log\n" +
				"| Number of requests | 100\n" +
				"| Average response size | 512\n" +
				"| 95th percentile of response size | 800\n" +
				"| Average requests per day | 10 |\n" +
				"|===\n\n" +

				"==== Requested Resources\n\n" +
				"[options=\"header\"]\n" +
				"|===\n" +
				"| Resource | Count\n" +
				"| `/index.html` | 50\n" +
				"| `/about.html` | 20\n" +
				"|===\n\n" +

				"==== Response Codes\n\n" +
				"[options=\"header\"]\n" +
				"|===\n" +
				"| Code | Name | Count\n" +
				"| 200 | OK | 80\n" +
				"| 404 | Not Found | 10\n" +
				"|===\n\n" +

				"==== Requesting addresses\n\n" +
				"[options=\"header\"]\n" +
				"|===\n" +
				"| Name | Count\n" +
				"| 192.168.1.1 | 30\n" +
				"| 10.0.0.2 | 20\n" +
				"|===\n",
		},
		{
			name: "Multiple files",
			info: &domain.FileInfo{
				Paths:             []string{"/var/log/nginx/access.log", "/var/log/nginx/access.log.1"},
				TotalRequests:     1000,
				AvgResponseSize:   1024,
				ResponseSize95p:   1500,
				AvgResponsePerDay: 100,
				FrequentURLs: []domain.URL{
					domain.NewURL("/home", 300),
					domain.NewURL("/login", 150),
					domain.NewURL("/dashboard", 100),
				},
				FrequentStatuses: []domain.Status{
					domain.NewStatus(200, 700),
					domain.NewStatus(403, 50),
					domain.NewStatus(500, 20),
				},
				FrequentAddresses: []domain.Address{
					domain.NewAddress("172.16.0.1", 200),
					domain.NewAddress("192.168.1.2", 150),
					domain.NewAddress("10.0.0.3", 120),
				},
			},
			expected: "==== General Information\n\n" +
				"[options=\"header\"]\n" +
				"|===\n" +
				"| Метрика | Значение\n" +
				"| Files | /var/log/nginx/access.log, /var/log/nginx/access.log.1\n" +
				"| Number of requests | 1000\n" +
				"| Average response size | 1024\n" +
				"| 95th percentile of response size | 1500\n" +
				"| Average requests per day | 100 |\n" +
				"|===\n\n" +
				"==== Requested Resources\n\n" +
				"[options=\"header\"]\n" +
				"|===\n" +
				"| Resource | Count\n" +
				"| `/home` | 300\n" +
				"| `/login` | 150\n" +
				"| `/dashboard` | 100\n" +
				"|===\n\n" +
				"==== Response Codes\n\n" +
				"[options=\"header\"]\n" +
				"|===\n" +
				"| Code | Name | Count\n" +
				"| 200 | OK | 700\n" +
				"| 403 | Forbidden | 50\n" +
				"| 500 | Internal Server Error | 20\n" +
				"|===\n\n" +
				"==== Requesting addresses\n\n" +
				"[options=\"header\"]\n" +
				"|===\n" +
				"| Name | Count\n" +
				"| 172.16.0.1 | 200\n" +
				"| 192.168.1.2 | 150\n" +
				"| 10.0.0.3 | 120\n" +
				"|===\n",
		},
		{
			name: "URL",
			info: &domain.FileInfo{
				Paths:             []string{"https://raw.githubusercontent.com/elastic/examples/master/Common%20Data%20Formats/nginx_logs/nginx_logs"},
				TotalRequests:     5000,
				AvgResponseSize:   2048,
				ResponseSize95p:   3000,
				AvgResponsePerDay: 500,
				FrequentURLs: []domain.URL{
					domain.NewURL("/home", 1000),
					domain.NewURL("/products", 800),
					domain.NewURL("/contact", 600),
				},
				FrequentStatuses: []domain.Status{
					domain.NewStatus(200, 4000),
					domain.NewStatus(404, 400),
					domain.NewStatus(503, 50),
				},
				FrequentAddresses: []domain.Address{
					domain.NewAddress("192.168.0.10", 500),
					domain.NewAddress("192.168.0.20", 450),
					domain.NewAddress("192.168.0.30", 300),
				},
			},
			expected: "==== General Information\n\n" +
				"[options=\"header\"]\n" +
				"|===\n" +
				"| Метрика | Значение\n" +
				"| Files | https://raw.githubusercontent.com/elastic/examples/master/Common%20Data%20Formats/nginx_logs/nginx_logs\n" +
				"| Number of requests | 5000\n" +
				"| Average response size | 2048\n" +
				"| 95th percentile of response size | 3000\n" +
				"| Average requests per day | 500 |\n" +
				"|===\n\n" +
				"==== Requested Resources\n\n" +
				"[options=\"header\"]\n" +
				"|===\n" +
				"| Resource | Count\n" +
				"| `/home` | 1000\n" +
				"| `/products` | 800\n" +
				"| `/contact` | 600\n" +
				"|===\n\n" +
				"==== Response Codes\n\n" +
				"[options=\"header\"]\n" +
				"|===\n" +
				"| Code | Name | Count\n" +
				"| 200 | OK | 4000\n" +
				"| 404 | Not Found | 400\n" +
				"| 503 | Service Unavailable | 50\n" +
				"|===\n\n" +
				"==== Requesting addresses\n\n" +
				"[options=\"header\"]\n" +
				"|===\n" +
				"| Name | Count\n" +
				"| 192.168.0.10 | 500\n" +
				"| 192.168.0.20 | 450\n" +
				"| 192.168.0.30 | 300\n" +
				"|===\n",
		},
	}

	logParser := parser.New()

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logParser.Adoc(tc.info, buf)

			assert.Equal(t, tc.expected, buf.String())
		})
	}
}
