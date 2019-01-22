package config

type Logging struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
}

type PProf struct {
	Addr    string `yaml:"addr"`
	Enabled bool   `yaml:"enabled"`
}

type Upload struct {
	Table string `yaml:"table"`
	DSN   string `yaml:"dsn"`
}

type CollectedLog struct {
	Tag        string `yaml:"tag"`
	Format     string `yaml:"format"`
	BufferSize int    `yaml:"buffer_size"`

	Transformers map[string]string `yaml:"transformers"`
	Upload       Upload            `yaml:"upload"`
}

type Processor struct {
	Workers int `yaml:"workers"`
}

type Receiver struct {
	Addr string `yaml:"addr"`
}

type Statsd struct {
	Addr    string `yaml:"addr"`
	Enabled bool   `yaml:"enabled"`
	Prefix  string `yaml:"prefix"`
}

type Backlog struct {
	Dir                       string `yaml:"dir"`
	MaxConcurrentHttpRequests int    `yaml:"max_concurrent_http_requests"`
}

type Config struct {
	Logging       Logging        `yaml:"logging"`
	PProf         PProf          `yaml:"pprof"`
	Processor     Processor      `yaml:"processor"`
	CollectedLogs []CollectedLog `yaml:"collected_logs"`
	Receiver      Receiver       `yaml:"receiver"`
	Statsd        Statsd         `yaml:"statsd"`
	Backlog       Backlog        `yaml:"backlog"`
	GoMaxProcs    int            `yaml:"gomaxprocs"`
}
