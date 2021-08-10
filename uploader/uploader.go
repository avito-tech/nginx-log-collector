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
	backlog     *backlog.Backlog
	tagContexts map[string]TagContext
	logger      zerolog.Logger
	metrics     *statsd.Client
	wg          *sync.WaitGroup
}

type TagContext struct {
	Config config.CollectedLog
	URL    string
}

func New(logs []config.CollectedLog, bl *backlog.Backlog, metrics *statsd.Client, logger *zerolog.Logger) (*Uploader, error) {
	tagContexts := make(map[string]TagContext)
	for _, l := range logs {
		uploadUrl, err := clickhouse.MakeUrl(l.Upload.DSN, l.Upload.Table, true, l.AllowErrorRatio)
		if err != nil {
			return nil, errors.Wrap(err, "unable to create uploader")
		}

		tagContexts[l.Tag] = TagContext{Config: l, URL: uploadUrl}
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	return &Uploader{
		tagContexts: tagContexts,
		wg:          wg,
		backlog:     bl,
		metrics:     metrics.Clone(statsd.Prefix("uploader")),
		logger:      logger.With().Str("component", "uploader").Logger(),
	}, nil
}

func (u *Uploader) Start(done <-chan struct{}, resultChan chan processor.Result) {
	defer u.wg.Done()
	u.logger.Info().Msg("starting")
	limiter := u.backlog.GetLimiter()
	isDone := false
	for result := range resultChan {
		tagContext, found := u.tagContexts[result.Tag]
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

		u.metrics.Gauge("upload_result_chan_len", len(resultChan))

		if isDone || len(resultChan) > maxResultChanLen {
			u.logger.Info().Msg("flushing to backlog")

			if tagContext.Config.Audit {
				// level is error because global log level is error
				u.logger.Error().Str("tag", result.Tag).Msgf("make new backlog job: %s", string(result.Data))
			}

			if err := u.backlog.MakeNewBacklogJob(tagContext.URL, result.Data); err != nil {
				u.logger.Fatal().Err(err).Msg("unable to create backlog job")
			}
			continue
		}

		limiter.Enter()
		u.wg.Add(1)
		go func(url string, data []byte, tag string, lines int) {
			tagTrimmed := tag[:len(tag)-1] // trim :

			u.metrics.Increment(fmt.Sprintf("uploading.batches.%s", tagTrimmed))
			u.metrics.Count(fmt.Sprintf("uploading.lines.%s", tagTrimmed), lines)

			err := clickhouse.Upload(url, data)
			if tagContext.Config.Audit {
				// level is error because global log level is error
				u.logger.Error().Str("tag", tag).Err(err).Msgf("upload: %s", string(data))
			}
			if err != nil {
				u.logger.Error().Str("tag", tag).Str("url", tagContext.URL).Err(err).Msg("upload error; creating backlog job")
				u.metrics.Increment("upload_error")
				if err := u.backlog.MakeNewBacklogJob(url, data); err != nil {
					u.logger.Fatal().Err(err).Msg("unable to create backlog job")
				}
				u.metrics.Increment(fmt.Sprintf("failed.batches.%s", tagTrimmed))
				u.metrics.Count(fmt.Sprintf("failed.lines.%s", tagTrimmed), lines)
			} else {
				u.metrics.Increment(fmt.Sprintf("ok.batches.%s", tagTrimmed))
				u.metrics.Count(fmt.Sprintf("ok.lines.%s", tagTrimmed), lines)
			}

			// old-style metric for compatibility
			u.metrics.Increment(fmt.Sprintf("upload_tag_%s_", tagTrimmed)) // trim :

			limiter.Leave()
			u.wg.Done()

		}(tagContext.URL, result.Data, result.Tag, result.Lines)
	}
	<-done
}

func (u *Uploader) Stop() {
	u.logger.Info().Msg("stopping")
	u.wg.Wait()
}
