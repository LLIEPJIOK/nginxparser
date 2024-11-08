package parser

import "time"

type log struct {
	RemoteAddress string
	RemoteUser    string
	TimeLocal     time.Time
	Method        string
	URL           string
	HTTPVersion   string
	Status        int
	BodyBytesSend int
	Referer       string
	UserAgent     string
}
