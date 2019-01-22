package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/alexcesaro/statsd.v2"
	"gopkg.in/yaml.v2"
	"nginx-log-collector/config"
	"nginx-log-collector/service"
)

// should be filled by go build
var Version = "0.0.0-devel"

func loadConfig(configFile string) *config.Config {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to read config file")
	}

	cfg := &config.Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Fatal().Err(err).Msg("unable to parse config")
	}
	return cfg
}

func setupLogger(cfg config.Logging) (*zerolog.Logger, error) {
	lvl, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil {
		return nil, err
	}
	zerolog.SetGlobalLevel(lvl)

	var z zerolog.Logger

	if cfg.Path != "" && cfg.Path != "stdout" {
		f, err := os.OpenFile(cfg.Path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return nil, err
		}
		z = zerolog.New(zerolog.SyncWriter(f))
	} else {
		z = zerolog.New(os.Stdout)
	}
	zt := z.With().Timestamp().Logger()
	return &zt, nil
}

func setupStatsD(cfg config.Statsd) (*statsd.Client, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	hostname = strings.Replace(hostname, ".", "_", -1)
	return statsd.New(
		statsd.Address(cfg.Addr),
		statsd.Prefix(cfg.Prefix+".host."+hostname),
		statsd.Mute(!cfg.Enabled),
	)
}

func main() {

	configFile := flag.String("config", "", "Config path")
	flag.Parse()

	if *configFile == "" {
		log.Fatal().Msg("-config flag should be set")
	}
	cfg := loadConfig(*configFile)

	logger, err := setupLogger(cfg.Logging)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to setup logging")
	}

	logger.Info().Str("version", Version).Msg("start")

	if cfg.GoMaxProcs > 0 {
		runtime.GOMAXPROCS(cfg.GoMaxProcs)
		logger.Info().Int("gomaxprocs", cfg.GoMaxProcs).Msg("gomaxprocs set")
	}

	metrics, err := setupStatsD(cfg.Statsd)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to setup statsd client")
	}

	done := make(chan struct{}, 1)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		logger.Info().Msg("got signal; exiting")
		close(done)
	}()

	s, err := service.New(cfg, metrics, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to init service")
	}

	if cfg.PProf.Enabled {
		logger.Info().Msg("starting pprof server")
		go func() {
			logger.Warn().Err(
				http.ListenAndServe(cfg.PProf.Addr, nil),
			).Msg("pprof server error")
		}()
	}

	s.Start(done)

	logger.Info().Msg("exit")
}
