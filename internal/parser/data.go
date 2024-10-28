package parser

import (
	"sync"
)

type data struct {
	mu            *sync.RWMutex
	paths         []string
	totalRequests int
	urls          map[string]int
	statuses      map[int]int
	sizeSum       int
	sizeSlice     []int
}

func newData() data {
	return data{
		mu:            &sync.RWMutex{},
		totalRequests: 0,
		urls:          make(map[string]int),
		statuses:      make(map[int]int),
		sizeSum:       0,
		sizeSlice:     make([]int, 0),
	}
}

func (d *data) processLog(logEntry *log) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.totalRequests++
	d.urls[logEntry.url]++
	d.statuses[logEntry.status]++
	d.sizeSum += logEntry.bodyBytesSend
	d.sizeSlice = append(d.sizeSlice, logEntry.bodyBytesSend)
}
