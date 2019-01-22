package processor

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gopkg.in/alexcesaro/statsd.v2"
	"nginx-log-collector/config"
)

const (
	flushInterval      = 30 * time.Second
	queueCheckInterval = 30 * time.Second
)

type Result struct {
	Tag  string
	Data []byte
}

type Processor struct {
	metrics *statsd.Client

	tagConverterMap  map[string]Converter
	tagBufferSizeMap map[string]int

	resultChan chan Result

	logger     zerolog.Logger
	wg         *sync.WaitGroup
	workersCnt int
}

func New(cfg config.Processor, logs []config.CollectedLog, metrics *statsd.Client, logger *zerolog.Logger) (*Processor, error) {

	tagConverterMap := make(map[string]Converter)
	tagBufferSizeMap := make(map[string]int)
	for _, l := range logs {
		converter, err := NewConverter(l)
		if err != nil {
			return nil, errors.Wrap(err, "unable to create converter")
		}
		if l.BufferSize <= 0 {
			return nil, fmt.Errorf("bad buffer size: %d for tag %s", l.BufferSize, l.Tag)

		}
		tagConverterMap[l.Tag] = converter
		tagBufferSizeMap[l.Tag] = l.BufferSize
	}

	return &Processor{
		tagConverterMap:  tagConverterMap,
		tagBufferSizeMap: tagBufferSizeMap,
		metrics:          metrics.Clone(statsd.Prefix("processor")),
		resultChan:       make(chan Result, 1000),
		wg:               &sync.WaitGroup{},
		workersCnt:       cfg.Workers,
		logger:           logger.With().Str("component", "processor").Logger(),
	}, nil
}

func (p *Processor) Start(msgChan <-chan []byte, done <-chan struct{}) {
	p.logger.Info().Msg("starting")
	for i := 0; i < p.workersCnt; i++ {
		p.wg.Add(1)
		go p.Worker(msgChan, done)
	}

	p.wg.Add(1)
	go p.queueMonitoring(done)
	<-done
	p.logger.Debug().Msg("got done")
}

func (p *Processor) Stop() {
	p.logger.Info().Msg("stopping")
	p.wg.Wait()
	p.logger.Debug().Msg("stopping [close phase]")
	close(p.resultChan)
}

func (p *Processor) ResultChan() chan Result {
	return p.resultChan
}

func (p *Processor) Worker(msgChan <-chan []byte, done <-chan struct{}) {
	defer p.wg.Done()

	var msg []byte
	var hostname, tag string

	tpMap := make(map[string]*tagProcessor)
	for tag := range p.tagConverterMap {
		bufferSize := p.tagBufferSizeMap[tag]
		tp := newTagProcessor(bufferSize, tag)
		tpMap[tag] = tp
		p.wg.Add(1)
		go tp.flusher(p.resultChan, done, p.wg)
	}

	for rawMsg := range msgChan {
		// format is defined in rsyslog
		s := bytes.SplitN(rawMsg, []byte{'\t'}, 3)
		if len(s) != 3 {
			p.logger.Warn().Bytes("msg", rawMsg).Int("len", len(s)).Msg("wrong message format")
			p.metrics.Increment("format_error")
			continue
		}

		hostname, tag, msg = string(s[0]), string(s[1]), s[2]

		converter, found := p.tagConverterMap[tag]
		if !found {
			p.logger.Warn().Str("host", hostname).Str("tag", tag).Msg("wrong tag")
			p.metrics.Increment("tag_error")
			continue
		}

		converted, err := converter.Convert(msg, hostname)
		if err != nil {
			p.logger.Warn().Str("host", hostname).Err(err).Msg("convert error")
			p.metrics.Increment("convert_error")
			continue
		}

		tp := tpMap[tag]
		tp.write(converted, p.resultChan)
	}

	for _, tp := range tpMap {
		tp.mu.Lock()
		tp.flush(p.resultChan)
		tp.mu.Unlock()
	}

	p.logger.Debug().Msg("processor worker done")
}

func (p *Processor) queueMonitoring(done <-chan struct{}) {
	defer p.wg.Done()
	ticker := time.NewTicker(queueCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.logger.Debug().Int("result_queue_len", len(p.resultChan)).Msg("queue stats")
			p.metrics.Count("result_queue_len", len(p.resultChan))
		case <-done:
			return
		}
	}
	p.logger.Debug().Msg("queueMonitoring exit")
}
