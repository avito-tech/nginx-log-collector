package processor

import (
	"bytes"
	"sync"
	"time"
)

type tagProcessor struct {
	mu          *sync.Mutex
	buffer      *bytes.Buffer
	nextFlushAt time.Time
	tag         string
	bufSize     int
	linesInBuf  int
}

func newTagProcessor(bufferSize int, tag string) *tagProcessor {
	b := make([]byte, 0, bufferSize)
	return &tagProcessor{
		buffer:      bytes.NewBuffer(b),
		nextFlushAt: time.Now().Add(flushInterval),
		tag:         tag,
		mu:          &sync.Mutex{},
		bufSize:     bufferSize,
		linesInBuf:  0,
	}
}

// XXX mutex should be taken
func (t *tagProcessor) flush(resultChan chan Result) {
	t.nextFlushAt = time.Now().Add(flushInterval)
	b := t.buffer.Bytes()
	if len(b) == 0 {
		return
	}
	clone := make([]byte, len(b)) // TODO sync pool?
	copy(clone, b)
	resultChan <- Result{
		Tag:   t.tag,
		Data:  clone,
		Lines: t.linesInBuf,
	}
	t.buffer.Reset()
	t.linesInBuf = 0
}

func (t *tagProcessor) writeLine(data []byte, resultChan chan Result) {
	t.mu.Lock()

	if t.buffer.Len()+len(data) > t.bufSize {
		t.flush(resultChan)
	}
	t.buffer.Write(data)
	t.linesInBuf += 1

	t.mu.Unlock()
}

func (t *tagProcessor) flusher(resultChan chan Result, done <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.mu.Lock()
			if time.Now().After(t.nextFlushAt) {
				t.flush(resultChan)
			}
			t.mu.Unlock()
		case <-done:
			return
		}
	}
}
