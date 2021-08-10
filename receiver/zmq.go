package receiver

/*

import (
	"sync"
	"time"

	"github.com/pebbe/zmq4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/alexcesaro/statsd.v2"
)

const (
	queueCheckInterval = 30 * time.Second
)

type ZmqReceiver struct {
	msgChan chan []byte
	socket  *zmq4.Socket
	metrics *statsd.Client
	wg      *sync.WaitGroup
	poller  *zmq4.Poller
}

func NewZmqReceiver(addr string, metrics *statsd.Client) (*ZmqReceiver, error) {
	socket, err := zmq4.NewSocket(zmq4.PULL)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init zmq socket")
	}
	err = socket.Bind(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to bind zmq socket with addr: %s", addr)
	}
	poller := zmq4.NewPoller()
	poller.Add(socket, zmq4.POLLIN)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	msgChan := make(chan []byte, 100000)
	return &ZmqReceiver{
		msgChan: msgChan,
		socket:  socket,
		metrics: metrics.Clone(statsd.Prefix("receiver.zmq")),
		wg:      wg,
		poller:  poller,
	}, nil
}

func (z *ZmqReceiver) MsgChan() chan []byte {
	return z.msgChan
}

func (z *ZmqReceiver) Start(done <-chan struct{}) {
	defer z.wg.Done()
	log.Info().Msg("zmq receiver starting")

	ticker := time.NewTicker(queueCheckInterval)
	defer ticker.Stop()
	z.wg.Add(1)
	go z.queueMonitoring(ticker.C, done)

	var cnt uint64
	for {
		select {
		case <-done:
			log.Debug().Msg("zmq done")
			return
		default:
			sockets, err := z.poller.Poll(time.Second)
			if err != nil {
				z.metrics.Increment("poll_errors")
				log.Warn().Err(err).Msg("zmq poll error; ignoring message")
				continue
			}
			if len(sockets) == 0 {
				continue
			}
			msg, err := sockets[0].Socket.RecvBytes(0)
			if err != nil {
				z.metrics.Increment("recv_errors")
				log.Warn().Err(err).Msg("zmq RecvBytes error; ignoring message")
				continue
			}
			z.msgChan <- msg
			cnt++
			if cnt%100 == 0 {
				log.Debug().Msg("100 done")
				z.metrics.Count("messages", cnt)
				cnt = 0
			}
		}
	}
}

func (z *ZmqReceiver) queueMonitoring(ticker <-chan time.Time, done <-chan struct{}) {
	defer z.wg.Done()
	for {
		select {
		case <-ticker:
			log.Debug().Int("msg_chan_len", len(z.msgChan)).Msg("queue stats")
			z.metrics.Count("msg_chan_len", len(z.msgChan))
		case <-done:
			return
		}
	}
	log.Debug().Msg("queueMonitoring exit")
}

func (z *ZmqReceiver) Stop() {
	log.Info().Msg("zmq receiver stopping")
	z.wg.Wait()
	z.poller.RemoveBySocket(z.socket)
	z.socket.Close()
	close(z.msgChan)
}
*/
