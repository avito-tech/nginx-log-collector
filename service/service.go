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
	httpReceiver *receiver.HttpReceiver
	tcpReceiver  *receiver.TCPReceiver
	processor    *processor.Processor
	uploader     *uploader.Uploader
	backlog      *backlog.Backlog

	logger  zerolog.Logger
	metrics *statsd.Client
}

func New(cfg *config.Config, metrics *statsd.Client, logger *zerolog.Logger) (*Service, error) {
	httpReceiver, err := receiver.NewHttpReceiver(&cfg.HttpReceiver, metrics, logger)
	if err != nil {
		return nil, errors.Wrap(err, "http receiver init error")
	}

	tcpReceiver, err := receiver.NewTCPReceiver(cfg.TCPReceiver.Addr, metrics, logger)
	if err != nil {
		return nil, errors.Wrap(err, "tcp receiver init error")
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
		httpReceiver: httpReceiver,
		tcpReceiver:  tcpReceiver,
		processor:    proc,
		uploader:     upl,
		backlog:      bl,
		logger:       logger.With().Str("component", "service").Logger(),
		metrics:      metrics.Clone(statsd.Prefix("service")),
	}, nil
}

func (s *Service) Start(done <-chan struct{}) {
	s.logger.Info().Msg("starting")

	sDone := make(chan struct{})
	if s.httpReceiver != nil {
		go s.httpReceiver.Start(sDone)
	}
	go s.tcpReceiver.Start(sDone)
	go s.processor.Start(sDone, s.httpReceiver.MsgChan(), s.tcpReceiver.MsgChan())
	go s.uploader.Start(sDone, s.processor.ResultChan())
	go s.backlog.Start(done)

	<-done
	close(sDone)

	s.logger.Info().Msg("stopping service")

	if s.httpReceiver != nil {
		s.httpReceiver.Stop()
		s.logger.Info().Msg("http receiver stopped")
	}

	s.tcpReceiver.Stop()
	s.logger.Info().Msg("tcp receiver stopped")

	s.processor.Stop()
	s.logger.Info().Msg("processor stopped")

	s.uploader.Stop()
	s.logger.Info().Msg("uploader stopped")

	s.backlog.Stop()
	s.logger.Info().Msg("backlog stopped")
}
