package receiver

import (
	"bufio"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gopkg.in/alexcesaro/statsd.v2"
)

const (
	readTimeout        = 30 * time.Second
	queueCheckInterval = 30 * time.Second
)

type TCPReceiver struct {
	msgChan  chan []byte
	listener *net.TCPListener

	metrics *statsd.Client
	logger  zerolog.Logger
	wg      *sync.WaitGroup
}

func NewTCPReceiver(addr string, metrics *statsd.Client, logger *zerolog.Logger) (*TCPReceiver, error) {
	resolvedAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "unable to resolve addr")
	}

	listener, err := net.ListenTCP("tcp", resolvedAddr)
	if err != nil {
		return nil, errors.Wrap(err, "unable to listen")
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	msgChan := make(chan []byte, 100000)
	return &TCPReceiver{
		msgChan:  msgChan,
		listener: listener,
		metrics:  metrics.Clone(statsd.Prefix("receiver.tcp")),
		wg:       wg,
		logger:   logger.With().Str("component", "receiver.tcp").Logger(),
	}, nil
}

func (t *TCPReceiver) MsgChan() chan []byte {
	return t.msgChan
}

func (t *TCPReceiver) Start(done <-chan struct{}) {
	t.logger.Info().Msg("starting")

	go t.queueMonitoring(done)

	defer t.listener.Close()
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-done:
				return
			default:
			}
			t.logger.Warn().Err(err).Msg("unable to accept connection")
			continue
		}
		t.wg.Add(1)
		go t.handle(conn, done)
	}
}

func (t *TCPReceiver) handle(conn net.Conn, done <-chan struct{}) {
	defer conn.Close()
	defer t.wg.Done()
	t.metrics.Increment("accepted")
	reader := bufio.NewReader(conn)
	var cnt uint64
	for {
		select {
		case <-done:
			return
		default:
		}
		err := conn.SetReadDeadline(time.Now().Add(readTimeout))
		if err != nil {
			t.logger.Warn().Err(err).Msg("set deadline error")
		}

		line, err := reader.ReadBytes('\n')

		if err != nil {
			if err == io.EOF {
				if len(line) > 0 {
					t.logger.Warn().Str("line", string(line)).Msg("unfinished line")
					t.metrics.Increment("line_error")
				}
			} else {
				t.logger.Debug().Err(err).Msg("read error (can be ignored)") // it's ok
			}
			break
		}
		t.msgChan <- line[:len(line)-1]
		cnt++
		if cnt%100 == 0 {
			t.metrics.Count("lines", 100)
		}
		if cnt%10000 == 0 {
			t.logger.Debug().Msg("10k lines processed")
			cnt = 0
		}
	}
}

func (t *TCPReceiver) queueMonitoring(done <-chan struct{}) {
	defer t.wg.Done()
	ticker := time.NewTicker(queueCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.logger.Debug().Int("msg_chan_len", len(t.msgChan)).Msg("queue stats")
			t.metrics.Count("msg_chan_len", len(t.msgChan))
		case <-done:
			return
		}
	}
	t.logger.Debug().Msg("queueMonitoring exit")
}

func (t *TCPReceiver) Stop() {
	t.listener.Close()
	t.logger.Info().Msg("stopping")
	t.wg.Wait()
	close(t.msgChan)
}
