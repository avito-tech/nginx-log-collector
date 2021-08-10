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
	Tag   string
	Data  []byte
	Lines int
}

type Processor struct {
	metrics *statsd.Client

	tagContexts map[string]TagContext

	resultChan chan Result

	logger     zerolog.Logger
	wg         *sync.WaitGroup
	workersCnt int
}

type TagContext struct {
	Config    config.CollectedLog
	Converter Converter
}

func New(cfg config.Processor, logs []config.CollectedLog, metrics *statsd.Client, logger *zerolog.Logger) (*Processor, error) {
	tagContexts := make(map[string]TagContext, len(logs))
	for _, l := range logs {
		converter, err := NewConverter(l)
		if err != nil {
			return nil, errors.Wrap(err, "unable to create converter")
		}
		if l.BufferSize <= 0 {
			return nil, fmt.Errorf("bad buffer size: %d for tag %s", l.BufferSize, l.Tag)

		}
		tagContexts[l.Tag] = TagContext{Config: l, Converter: converter}
	}

	return &Processor{
		tagContexts: tagContexts,
		metrics:     metrics.Clone(statsd.Prefix("processor")),
		resultChan:  make(chan Result, 1000),
		wg:          &sync.WaitGroup{},
		workersCnt:  cfg.Workers,
		logger:      logger.With().Str("component", "processor").Logger(),
	}, nil
}

func (p *Processor) Start(done <-chan struct{}, msgChanList ...chan []byte) {
	p.logger.Info().Msg("starting")
	for i := 0; i < p.workersCnt; i++ {
		p.wg.Add(1)
		go p.Worker(done, p.aggregateChan(msgChanList))
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

func (p *Processor) Worker(done <-chan struct{}, msgChan <-chan []byte) {
	defer p.wg.Done()

	tpMap := make(map[string]*tagProcessor)
	for tag, tagContext := range p.tagContexts {
		tp := newTagProcessor(tagContext.Config.BufferSize, tag)
		tpMap[tag] = tp
		p.wg.Add(1)
		go tp.flusher(p.resultChan, done, p.wg)
	}

	for rawMsg := range msgChan {
		// format is defined in rsyslog
		s := bytes.SplitN(rawMsg, []byte{'\t'}, 3)
		if len(s) != 3 {
			p.logger.Error().Bytes("msg", rawMsg).Int("len", len(s)).Msg("wrong message format")
			p.metrics.Increment("format_error")
			continue
		}

		hostname, tag, msg := string(s[0]), string(s[1]), s[2]
		tagContext, found := p.tagContexts[tag]
		if !found {
			p.logger.Warn().Str("host", hostname).Str("tag", tag).Msg("wrong tag")
			p.metrics.Increment("tag_error")
			continue
		}

		converted, err := tagContext.Converter.Convert(msg, hostname)
		if err != nil {
			logEvent := p.logger.Error().Str("host", hostname).Err(err)
			// AD-17284: always log the message
			logEvent = logEvent.Bytes("msg", msg)

			logEvent.Msg("convert error")
			p.metrics.Increment("convert_error")
			continue
		}

		tp := tpMap[tag]
		tp.writeLine(converted, p.resultChan)
		if tagContext.Config.Audit {
			p.logger.Error().Str("tag", tag).Msgf("write to buffer: %s", string(converted))
		}
	}

	for _, tp := range tpMap {
		tp.mu.Lock()
		tp.flush(p.resultChan)
		tp.mu.Unlock()
	}

	p.logger.Debug().Msg("processor worker done")
}

// aggregateChan aggregates list of channels to single channel
func (p *Processor) aggregateChan(msgChanList []chan []byte) chan []byte {
	bufferSize := 0
	for _, msgChan := range msgChanList {
		bufferSize += cap(msgChan)
	}

	aggregate := make(chan []byte, bufferSize)
	var wg sync.WaitGroup
	wg.Add(len(msgChanList))
	for _, msgChan := range msgChanList {
		go func(msgChan <-chan []byte) {
			for msg := range msgChan {
				aggregate <- msg
			}
			wg.Done()
		}(msgChan)
	}
	go func() {
		wg.Wait()
		close(aggregate)
	}()

	return aggregate
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
			p.logger.Debug().Msg("queueMonitoring exit")
			return
		}
	}
}
