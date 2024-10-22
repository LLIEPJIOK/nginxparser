package parser

import "time"

type log struct {
	remoteAddress string
	remoteUser    string
	timeLocal     time.Time
	method        string
	url           string
	httpVersion   string
	status        int
	bodyBytesSend int
	referer       string
	userAgent     string
}
