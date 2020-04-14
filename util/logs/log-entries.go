package logs

import (
	"time"
)

type logEntry struct {
	timestamp time.Time
	podName   string
	content   string
}

type logEntries []logEntry

func (l logEntries) Len() int {
	return len(l)
}

func (l logEntries) Less(i, j int) bool {
	return l[i].timestamp.Before(l[j].timestamp)
}

func (l logEntries) Swap(i, j int) {
	tmp := l[i]
	l[i] = l[j]
	l[j] = tmp
}
