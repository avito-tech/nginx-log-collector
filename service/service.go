package service

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gopkg.in/alexcesaro/statsd.v2"
	"nginx-log-collector/backlog"
	"nginx-log-collector/config"
	"nginx-log-collector/processor"
	"nginx-log-collector/receiver"
	"nginx-log-collector/uploader"
)

type Service struct {
	receiver  *receiver.TCPReceiver
	processor *processor.Processor
	uploader  *uploader.Uploader
	backlog   *backlog.Backlog

	logger  zerolog.Logger
	metrics *statsd.Client
}

func New(cfg *config.Config, metrics *statsd.Client, logger *zerolog.Logger) (*Service, error) {

	recv, err := receiver.NewTCPReceiver(cfg.Receiver.Addr, metrics, logger)
	if err != nil {
		return nil, errors.Wrap(err, "receiver init error")
	}

	proc, err := processor.New(cfg.Processor, cfg.CollectedLogs, metrics, logger)
	if err != nil {
		return nil, errors.Wrap(err, "processor init error")
	}

	bl, err := backlog.New(cfg.Backlog, metrics, logger)
	if err != nil {
		return nil, errors.Wrap(err, "backlog init error")
	}

	upl, err := uploader.New(cfg.CollectedLogs, bl, metrics, logger)
	if err != nil {
		return nil, errors.Wrap(err, "uploader init error")
	}

	return &Service{
		receiver:  recv,
		backlog:   bl,
		processor: proc,
		uploader:  upl,
		metrics:   metrics.Clone(statsd.Prefix("service")),
		logger:    logger.With().Str("component", "service").Logger(),
	}, nil
}

func (s *Service) Start(done <-chan struct{}) {
	s.logger.Info().Msg("starting")
	msgChan := s.receiver.MsgChan()
	resultChan := s.processor.ResultChan()

	sDone := make(chan struct{})

	go s.receiver.Start(sDone)
	go s.processor.Start(msgChan, sDone)
	go s.uploader.Start(resultChan, sDone)
	go s.backlog.Start(done)

	<-done
	close(sDone)

	s.logger.Info().Msg("stopping service")
	s.receiver.Stop()
	s.logger.Info().Msg("receiver stopped")
	s.processor.Stop()
	s.logger.Info().Msg("processor stopped")
	s.uploader.Stop()
	s.logger.Info().Msg("uploader stopped")
	s.backlog.Stop()
	s.logger.Info().Msg("backlog stopped")
}
