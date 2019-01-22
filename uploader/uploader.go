package uploader

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gopkg.in/alexcesaro/statsd.v2"
	"nginx-log-collector/backlog"
	"nginx-log-collector/clickhouse"
	"nginx-log-collector/config"
	"nginx-log-collector/processor"
)

const maxResultChanLen = 10

type Uploader struct {
	backlog   *backlog.Backlog
	tagURLMap map[string]string
	logger    zerolog.Logger
	metrics   *statsd.Client
	wg        *sync.WaitGroup
}

func New(logs []config.CollectedLog, bl *backlog.Backlog, metrics *statsd.Client, logger *zerolog.Logger) (*Uploader, error) {
	tagURLMap := make(map[string]string)
	for _, l := range logs {
		uploadUrl, err := clickhouse.MakeUrl(l.Upload.DSN, l.Upload.Table)
		if err != nil {
			return nil, errors.Wrap(err, "unable to create uploader")
		}

		tagURLMap[l.Tag] = uploadUrl
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	return &Uploader{
		tagURLMap: tagURLMap,
		wg:        wg,
		backlog:   bl,
		metrics:   metrics.Clone(statsd.Prefix("uploader")),
		logger:    logger.With().Str("component", "uploader").Logger(),
	}, nil
}

func (u *Uploader) Start(resultChan chan processor.Result, done <-chan struct{}) {
	defer u.wg.Done()
	u.logger.Info().Msg("starting")
	limiter := u.backlog.GetLimiter()
	isDone := false
	for result := range resultChan {
		uploadUrl, found := u.tagURLMap[result.Tag]
		if !found {
			u.metrics.Increment("tag_missing_error")
			u.logger.Warn().Str("tag", result.Tag).Msg("tag missing in uploader")
			continue
		}

		if !isDone {
			select {
			case <-done:
				isDone = true
			default:
			}
		}

		if isDone || len(resultChan) > maxResultChanLen {
			u.logger.Info().Msg("flushing to backlog")
			if err := u.backlog.MakeNewBacklogJob(uploadUrl, result.Data); err != nil {
				u.logger.Fatal().Err(err).Msg("unable to create backlog job")
			}
			continue
		}

		limiter.Enter()
		u.wg.Add(1)
		go func(url string, data []byte, tag string) {
			if err := clickhouse.Upload(url, data); err != nil {
				u.logger.Warn().Str("tag", tag).Str("url", uploadUrl).Err(err).Msg("upload error; creating backlog job")
				u.metrics.Increment("upload_error")
				if err := u.backlog.MakeNewBacklogJob(url, data); err != nil {
					u.logger.Fatal().Err(err).Msg("unable to create backlog job")
				}
			}
			u.metrics.Increment(fmt.Sprintf("upload_tag_%s_", tag[:len(tag)-1])) // trim :
			limiter.Leave()
			u.wg.Done()

		}(uploadUrl, result.Data, result.Tag)
	}
	<-done
}

func (u *Uploader) Stop() {
	u.logger.Info().Msg("stopping")
	u.wg.Wait()
}
