package parser

import (
	"sync"
)

const timeLayout = "02/Jan/2006"

type data struct {
	mu             *sync.RWMutex
	paths          []string
	totalRequests  int
	urls           map[string]int
	statuses       map[int]int
	sizeSum        int
	sizeSlice      []int
	addresses      map[string]int
	requestsPerDay map[string]int
}

func newData() data {
	return data{
		mu:             &sync.RWMutex{},
		paths:          make([]string, 0),
		totalRequests:  0,
		urls:           make(map[string]int),
		statuses:       make(map[int]int),
		sizeSum:        0,
		sizeSlice:      make([]int, 0),
		addresses:      make(map[string]int),
		requestsPerDay: make(map[string]int),
	}
}

func (d *data) processLog(logEntry *log) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.totalRequests++
	d.urls[logEntry.URL]++
	d.statuses[logEntry.Status]++
	d.sizeSum += logEntry.BodyBytesSend
	d.sizeSlice = append(d.sizeSlice, logEntry.BodyBytesSend)
	d.addresses[logEntry.RemoteAddress]++
	d.requestsPerDay[logEntry.TimeLocal.Format(timeLayout)]++
}
