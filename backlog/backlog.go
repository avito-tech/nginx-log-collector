package backlog

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"nginx-log-collector/clickhouse"
	"nginx-log-collector/config"
	"nginx-log-collector/utils"
	"gopkg.in/alexcesaro/statsd.v2"
)

const (
	backlogSuffix             = ".backlog"
	writeSuffix               = ".writing"
	checkInterval             = 30 * time.Second
	maxConcurrentHttpRequests = 32
)

type Backlog struct {
	dir string

	logger  zerolog.Logger
	metrics *statsd.Client
	makeMu  *sync.Mutex
	wg      *sync.WaitGroup
	limiter utils.Limiter
}

func New(cfg config.Backlog, metrics *statsd.Client, logger *zerolog.Logger) (*Backlog, error) {
	err := os.MkdirAll(cfg.Dir, 0755)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create backlog directory")
	}
	files, err := ioutil.ReadDir(cfg.Dir)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read backlog directory")
	}
	for _, f := range files {
		fName := f.Name()
		if strings.HasSuffix(fName, writeSuffix) {
			// remove incomplete files
			path := filepath.Join(cfg.Dir, fName)
			err = os.Remove(path)
			if err != nil {
				return nil, errors.Wrap(err, "unable to remove incomplete file")
			}
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	requestsLimit := maxConcurrentHttpRequests
	if cfg.MaxConcurrentHttpRequests > 0 {
		requestsLimit = cfg.MaxConcurrentHttpRequests

	}

	return &Backlog{
		dir:     cfg.Dir,
		makeMu:  &sync.Mutex{},
		wg:      wg,
		metrics: metrics.Clone(statsd.Prefix("backlog")),
		logger:  logger.With().Str("component", "backlog").Logger(),
		limiter: utils.NewLimiter(requestsLimit),
	}, nil
}

func (b *Backlog) Start(done <-chan struct{}) {
	b.logger.Info().Msg("starting")
	defer b.wg.Done()

	// don't wait for the first tick
	b.check(done)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			b.check(done)
		}
	}
}

func (b *Backlog) Stop() {
	b.logger.Info().Msg("stopping")
	b.wg.Wait()
}

func (b *Backlog) processFile(filename string) {
	b.logger.Info().Str("file", filename).Msg("starting backlog job")
	b.metrics.Increment("job_start")
	path := filepath.Join(b.dir, filename)
	file, err := os.Open(path)
	if err != nil {
		b.logger.Error().Err(err).Msg("unable to open backlog file")
		b.metrics.Increment("open_error")
		return
	}
	if !checkCrc(file) {
		file.Close()
		b.logger.Error().Msg("invalid crc32 checksum")
		if err = os.Remove(path); err != nil {
			b.logger.Fatal().Err(err).Msg("unable to remove invalid backlog file")
			b.metrics.Increment("remove_error")
		}
		return
	}

	file.Seek(4, 0) // crc offset
	url := readUrl(file)

	err = clickhouse.UploadReader(url, file)

	file.Close()

	if err != nil {
		b.logger.Error().Err(err).Msg("unable to upload backlog file")
		b.metrics.Increment("upload_error")
	} else {
		if err = os.Remove(path); err != nil {
			b.logger.Fatal().Err(err).Msg("unable to remove finished backlog file")
			b.metrics.Increment("upload_remove_error")
		}
	}

}

func (b *Backlog) check(done <-chan struct{}) {
	b.logger.Debug().Msg("starting backlog check")
	files, err := ioutil.ReadDir(b.dir)
	if err != nil {
		b.logger.Error().Err(err).Msg("unable to read backlog directory")
		return
	}

	wg := &sync.WaitGroup{}
	for _, f := range files {
		select {
		case <-done:
			return
		default:
		}
		if !strings.HasSuffix(f.Name(), backlogSuffix) {
			continue
		}

		b.limiter.Enter()
		wg.Add(1)
		go func(name string) {
			b.processFile(name)
			b.limiter.Leave()
			wg.Done()
		}(f.Name())
	}
	wg.Wait()
	return
}

func (b *Backlog) MakeNewBacklogJob(url string, data []byte) error {
	b.makeMu.Lock()
	defer b.makeMu.Unlock()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	file, err := ioutil.TempFile(b.dir, timestamp+"_*"+writeSuffix)
	if err != nil {
		b.metrics.Increment("tmp_file_create_error")
		return errors.Wrap(err, "unable to create tmp file")
	}
	defer file.Close()

	serializedUrl := serializeString(url)
	crcBuf := calcCrc(serializedUrl, data)

	file.Write(crcBuf)
	file.Write(serializedUrl)
	file.Write(data)

	file.Sync()

	if err := b.Rename(file.Name()); err != nil {
		b.metrics.Increment("tmp_file_rename_error")
		return errors.Wrap(err, "unable to finish backlog job")
	}
	b.metrics.Increment("backlog_job_created")
	return nil
}

func (b *Backlog) Rename(oldPath string) error {
	newPath := baseFileName(oldPath) + backlogSuffix
	return os.Rename(oldPath, newPath)
}

func (b *Backlog) GetLimiter() utils.Limiter {
	return b.limiter
}

func serializeString(s string) []byte {
	b := make([]byte, len(s)+4)
	binary.BigEndian.PutUint32(b, uint32(len(s)))
	copy(b[4:], s)
	return b
}

func calcCrc(serializedUrl []byte, data []byte) []byte {
	h := crc32.NewIEEE()
	h.Write(serializedUrl)
	h.Write(data)
	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, h.Sum32())
	return crcBuf
}

func readUrl(file *os.File) string {
	strLenBuf := make([]byte, 4)
	file.Read(strLenBuf)
	strLen := binary.BigEndian.Uint32(strLenBuf)

	urlBuf := make([]byte, strLen)
	file.Read(urlBuf)
	return string(urlBuf)
}

func checkCrc(file *os.File) bool {
	crcBuf := make([]byte, 4)
	file.Read(crcBuf)
	expectedCrc := binary.BigEndian.Uint32(crcBuf)
	h := crc32.NewIEEE()

	io.Copy(h, file)
	return expectedCrc == h.Sum32()
}

func baseFileName(path string) string {
	for i := len(path) - 1; i >= 0 && !os.IsPathSeparator(path[i]); i-- {
		if path[i] == '.' {
			return path[:i]
		}
	}
	return path
}
