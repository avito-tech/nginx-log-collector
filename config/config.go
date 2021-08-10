package config

import (
	"nginx-log-collector/processor/functions"
)

type Backlog struct {
	Dir                       string `yaml:"dir"`
	MaxConcurrentHttpRequests int    `yaml:"max_concurrent_http_requests"`
}

type CollectedLog struct {
	Tag             string `yaml:"tag"`
	Format          string `yaml:"format"`
	AllowErrorRatio int    `yaml:"allow_error_ratio"`
	BufferSize      int    `yaml:"buffer_size"`

	Transformers   functions.FunctionSignatureMap `yaml:"transformers"`
	Upload         Upload                         `yaml:"upload"`

	Audit bool `yaml:"audit"` // debug feature
}

type HttpReceiver struct {
	Enabled bool   `yaml:"enabled"`
	Url     string `yaml:"url"`
}

type Logging struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
}

type PProf struct {
	Addr    string `yaml:"addr"`
	Enabled bool   `yaml:"enabled"`
}

type Processor struct {
	Workers int `yaml:"workers"`
}

type Statsd struct {
	Addr    string `yaml:"addr"`
	Enabled bool   `yaml:"enabled"`
	Prefix  string `yaml:"prefix"`
}

type TCPReceiver struct {
	Addr string `yaml:"addr"`
}

type Upload struct {
	Table string `yaml:"table"`
	DSN   string `yaml:"dsn"`
}

type Config struct {
	Backlog       Backlog        `yaml:"backlog"`
	CollectedLogs []CollectedLog `yaml:"collected_logs"`
	HttpReceiver  HttpReceiver   `yaml:"httpReceiver"`
	Logging       Logging        `yaml:"logging"`
	PProf         PProf          `yaml:"pprof"`
	Processor     Processor      `yaml:"processor"`
	TCPReceiver   TCPReceiver    `yaml:"tcpReceiver"`
	Statsd        Statsd         `yaml:"statsd"`
	GoMaxProcs    int            `yaml:"gomaxprocs"`
}
