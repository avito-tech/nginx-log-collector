package receiver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/alexcesaro/statsd.v2"

	"nginx-log-collector/config"
	"nginx-log-collector/utils"
)

type HttpReceiver struct {
	config  *config.HttpReceiver
	metrics *statsd.Client
	msgChan chan []byte
	logger  zerolog.Logger
	wg      *sync.WaitGroup
}

type processedLogEntry struct {
	EventDateTime string `json:"event_datetime"`
	EventDate     string `json:"event_date"`
	Hostname      string `json:"hostname"`
	Message       string `json:"message"`
	SetupID       string `json:"request_id"`
	Severity      string `json:"severity"`
	User          string `json:"user"`
	RowNumber     int    `json:"row_number"`
}

const (
	httpReadTimeout     = 30 * time.Second
	httpWriteTimeout    = 5 * time.Second
	httpShutdownTimeout = 5 * time.Second

	dateFormat           = "2006-01-02"
	headerHostname       = "X-Log-Source"
	headerSetupID        = "X-Setup-Id"
	multipartFormMaxSize = 100 * 1024 * 1024
	tag                  = "puppet:"
)

var (
	datetimeTransformers = []*utils.DatetimeTransformer{
		{
			"2006-01-02 15:04:05 -0700",
			time.RFC3339,
			time.Local,
		},
		{
			"2006-01-02 15:04:05 0700",
			time.RFC3339,
			time.Local,
		},
	}
)

func NewHttpReceiver(cfg *config.HttpReceiver, metrics *statsd.Client, logger *zerolog.Logger) (*HttpReceiver, error) {
	httpReceiver := &HttpReceiver{
		config:  cfg,
		metrics: metrics.Clone(statsd.Prefix("receiver.http")),
		msgChan: make(chan []byte, 100000),
		wg:      &sync.WaitGroup{},
		logger:  logger.With().Str("component", "receiver.http").Logger(),
	}
	return httpReceiver, nil
}

func (h *HttpReceiver) MsgChan() chan []byte {
	return h.msgChan
}

func (h *HttpReceiver) Start(done <-chan struct{}) {
	h.logger.Info().Msg("Starting")

	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		h.handle(w, r)
	})

	server := &http.Server{
		Addr:         h.config.Url,
		Handler:      router,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
	}

	h.wg.Add(1)
	go h.gracefulShutdownSentry(done, server)

	h.wg.Add(1)
	go h.queueStats(done)

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		h.logger.Fatal().Err(err).Msgf("Could not listen on %s", h.config.Url)
		return
	}
}

func (h *HttpReceiver) Stop() {
	h.logger.Info().Msg("stopping")
	h.wg.Wait()
	close(h.msgChan)
}

// gracefulShutdownSentry tries to gracefully shutdown http server once done channel is closed
func (h *HttpReceiver) gracefulShutdownSentry(done <-chan struct{}, server *http.Server) {
	defer h.wg.Done()

	<-done
	h.logger.Warn().Msgf("Http server is shutting down with time of %.2f seconds", httpShutdownTimeout.Seconds())

	ctx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
	defer cancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil {
		h.logger.Error().Msgf("Failed to shutdown gracefully: %v", err)
	}
}

// handle processes single request
func (h *HttpReceiver) handle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	hostname := r.Header.Get(headerHostname)
	if hostname == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Missing or empty " + headerHostname + " header"))
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "<empty>"
	}

	setupID := r.Header.Get(headerSetupID)

	if strings.HasPrefix(contentType, "multipart/form-data") {
		err := r.ParseMultipartForm(multipartFormMaxSize)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to parse form")

			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Failed to parse form: " + err.Error()))
			return
		}

		for _, fileHeaders := range r.MultipartForm.File {
			for _, fileHeader := range fileHeaders {
				file, err := fileHeader.Open()
				if err != nil {
					h.logger.Error().Err(err).Str("filename", fileHeader.Filename).Msg("Failed to open the file")
					continue
				}

				h.processContent(file, hostname, setupID)
				_ = file.Close()
			}
		}
	} else if strings.HasPrefix(contentType, "text/plain") {
		h.processContent(r.Body, hostname, setupID)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Only 'text/plain' and 'multipart/form-data' content types are supported, got " + contentType))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// processContent processes request body or posted file
func (h *HttpReceiver) processContent(content io.Reader, hostname, setupID string) {
	hasData := true
	reader := bufio.NewReader(content)
	rowNumber := 0

	var (
		logEntryCurr *processedLogEntry
		logEntryNext *processedLogEntry
	)

	for hasData {
		line, err := reader.ReadString('\n')
		hasData = err != io.EOF
		if err != nil && hasData {
			h.logger.Error().Err(err).Str("line", line).Msg("Got an error while reading the line")
		}

		line = strings.TrimSuffix(line, "\n")
		logEntryNext, err = h.transformLine(line)

		if logEntryCurr != nil {
			if logEntryNext == nil {
				logEntryCurr.Message += "\n" + line
			}
		}

		if logEntryNext != nil { // new log entry is up
			if logEntryCurr != nil { // flush previous log entry
				h.sendLogEntry(logEntryCurr)
			}

			// switch to the new one
			logEntryCurr = logEntryNext
			logEntryCurr.Hostname = hostname
			logEntryCurr.RowNumber = rowNumber
			logEntryCurr.SetupID = setupID
			rowNumber++
		} else if logEntryCurr != nil { // another row of multiline log entry
			logEntryCurr.Message += "\n" + line
		} else if err != nil {
			h.logger.Error().Err(err).Str("line", line).Msg("Got an error while processing the line")
		}

		if !hasData && logEntryCurr != nil { // also flush the last log entry
			h.sendLogEntry(logEntryCurr)
		}
	}
}

// queueStats sends metric with length of message once per amount of time
func (h *HttpReceiver) queueStats(done <-chan struct{}) {
	defer h.wg.Done()

	ticker := time.NewTicker(queueCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			h.logger.Debug().Msg("queueStats stopped")
			return
		case <-ticker.C:
			queueSize := len(h.msgChan)
			h.logger.Debug().Int("http_msg_chan_len", queueSize).Msg("http queue stats")
			h.metrics.Count("http_msg_chan_len", queueSize)
		}
	}
}

// sendLogEntry prepares the bytes buffer and sends it
func (h *HttpReceiver) sendLogEntry(logEntry *processedLogEntry) {
	data, err := json.Marshal(logEntry)
	if err != nil {
		h.logger.Error().Err(err).Str("logEntry", fmt.Sprintf("%+v", logEntry)).Msg("Failed to marshal log entry")
		return
	}

	buffer := bytes.Buffer{}
	buffer.WriteString(logEntry.Hostname)
	buffer.WriteByte('\t')
	buffer.WriteString(tag)
	buffer.WriteByte('\t')
	buffer.Write(data)

	h.msgChan <- buffer.Bytes()
}

// transformLine parses log line and transforms it to access log format
// expected line format:
// 2020-04-24 18:14:42 +0300 Puppet (info): Applying configuration version '1587741267'
func (h *HttpReceiver) transformLine(line string) (*processedLogEntry, error) {
	// "2020-04-24", "18:14:42", "+0300", "Puppet", "(info):", "Applying configuration version '1587741267'"
	parts := strings.SplitN(line, " ", 6)
	if len(parts) != 6 {
		return nil, fmt.Errorf("wrong line structure")
	}

	// "(info):"
	severity := parts[4]
	if !strings.HasPrefix(severity, "(") || !strings.HasSuffix(severity, "):") {
		return nil, fmt.Errorf("wrong severity format: %s", severity)
	}
	severity = severity[1 : len(severity)-2]

	// "2020-04-24", "18:14:42", "+0300"
	datetime := strings.Join(parts[0:3], " ")
	parsed, transformer, err := utils.TryDatetimeFormats(datetime, datetimeTransformers)
	if err != nil {
		return nil, err
	}

	// "Puppet"
	user := parts[3]
	if user == "" {
		return nil, fmt.Errorf("wrong user format: %s", user)
	}

	// "Applying configuration version '1587741267'"
	message := parts[5] // message can be empty

	return &processedLogEntry{
		EventDateTime: parsed.Format(transformer.FormatDst),
		EventDate:     parsed.Format(dateFormat),
		Message:       message,
		Severity:      severity,
		User:          user,
	}, nil
}
