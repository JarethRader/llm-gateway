package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"packages/lib/golang/shared/config"
	"sync"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/dto"
)

type Relay struct {
	cfg config.SSEStreaming
	lgr *slog.Logger
}

func New(cfg config.SSEStreaming, lgr *slog.Logger) *Relay {
	return &Relay{
		cfg: cfg,
		lgr: lgr,
	}
}

var bufferPool = sync.Pool{New: func() any { b := make([]byte, 32*1024); return &b }}

// writeOp carries data or a signal to the write process
type writeOp struct {
	data []byte
	done chan error
}

// readCloser wraps a source context and calls Close() when
// the context is cancelled, which unblocks the blocking read
type readCloser struct {
	io.ReadCloser
	cancel func()
}

func (r readCloser) Close() error {
	r.cancel()
	return r.ReadCloser.Close()
}

type readResult struct {
	frame []byte
	err   error
}

// sseFrameReader unifies raw-read and frame-aware-read into one interface
type sseFrameReader struct {
	src        io.ReadCloser
	br         *bufio.Reader
	buf        []byte
	frameAware bool
}

func (fr *sseFrameReader) ReadFrame() ([]byte, error) {
	if fr.frameAware {
		return readSSEFrame(fr.br)
	}
	n, err := fr.src.Read(fr.buf)
	if n > 0 {
		cp := make([]byte, n)
		copy(cp, fr.buf[:n])
		return cp, err
	}
	return nil, err
}

func (p *Relay) RelaySSE(
	ctx context.Context,
	dispatchStart time.Time,
	src io.ReadCloser,
	w http.ResponseWriter,
	flusher http.Flusher,
	cfg config.SSEStreaming,
) dto.RelayResult {
	upstreamCtx, cancelUpstream := context.WithCancel(ctx)
	defer cancelUpstream()

	reader := readCloser{
		src,
		func() { cancelUpstream() },
	}
	defer reader.Close()

	var result dto.RelayResult
	firstByte := true

	writeCh := make(chan writeOp, 16)
	var writeWg sync.WaitGroup
	writeWg.Add(1)
	go func() {
		defer writeWg.Done()
		for op := range writeCh {
			_, err := w.Write(op.data)
			if err == nil {
				flusher.Flush()
			}
			if op.done != nil {
				op.done <- err
				op.done = nil
			}
		}
	}()

	var heartbeatWg sync.WaitGroup
	if cfg.HeartbeatInterval > 0 {
		heartbeatWg.Add(1)
		go func() {
			defer heartbeatWg.Done()
			ticker := time.NewTicker(cfg.HeartbeatInterval)
			defer ticker.Stop()
			for {
				select {
				case <-upstreamCtx.Done():
					return
				case <-ticker.C:
					op := writeOp{data: []byte(": heartbeat\n\n")}
					select {
					case writeCh <- op:
					case <-upstreamCtx.Done():
					}
				}
			}
		}()
	}

	watchdog := time.AfterFunc(cfg.IdleTimeout, func() { cancelUpstream() })
	defer watchdog.Stop()

	fr := &sseFrameReader{
		src:        reader,
		frameAware: cfg.FrameAware,
	}
	if cfg.FrameAware {
		fr.br = bufio.NewReaderSize(reader, 32*1024)
	} else {
		p := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(p)
		fr.buf = *p
	}

	readCh := make(chan readResult, 1)
	var readWg sync.WaitGroup
	readWg.Add(1)

	go func() {
		defer readWg.Done()
		for {
			frame, err := fr.ReadFrame()
			select {
			case readCh <- readResult{frame, err}:
			default: // main loop exited, don't block
			}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case res := <-readCh:
			frame, err := res.frame, res.err

			if len(frame) > 0 {
				if firstByte {
					result.TTFTMS = time.Since(dispatchStart).Seconds() * 1000
					firstByte = false
				}

				op := writeOp{
					data: frame,
					done: make(chan error, 1),
				}
				select {
				case writeCh <- op:
					werr := <-op.done
					if werr != nil {
						return completeResult(result, "client_gone", werr, writeCh, &writeWg, &heartbeatWg)
					}
					result.Bytes += int64(len(frame))
					watchdog.Reset(cfg.IdleTimeout)

					if cfg.FrameAware {
						if p, c, ok := parseUsage(frame); ok {
							result.PromptTokens, result.CompletionTokens = p, c
						}
						if isDone(frame) {
							return completeResult(result, "done", nil, writeCh, &writeWg, &heartbeatWg)
						}
					}
				case <-upstreamCtx.Done():
					return completeResult(result, "idle_timeout", upstreamCtx.Err(), writeCh, &writeWg, &heartbeatWg)
				}
			}

			if err == io.EOF {
				result.EndReason = "eof"
				goto done
			}
			if err != nil {
				emitErrorEvent(w, flusher, err)
				result.EndReason = classifyReadErr(upstreamCtx)
				result.Err = err
				goto done
			}
		case <-upstreamCtx.Done():
			goto done
		}
	}

done:
	cancelUpstream()
	reader.Close()
	readWg.Wait()
	return completeResult(result, result.EndReason, result.Err, writeCh, &writeWg, &heartbeatWg)
}

// completeResult closes the write channel, waits for goroutines to drain, and returns the final result.
func completeResult(r dto.RelayResult, endReason string, err error, writeCh chan writeOp, writeWg, heartbeatWg *sync.WaitGroup) dto.RelayResult {
	close(writeCh)
	writeWg.Wait()
	heartbeatWg.Wait()
	r.EndReason = endReason
	r.Err = err
	return r
}

// writeSSEFrame sends an SSE-formatted frame to the client
func emitErrorEvent(w http.ResponseWriter, flusher http.Flusher, err error) {
	msg := map[string]any{
		"error": map[string]string{
			"message": err.Error(),
		},
	}
	data, _ := json.Marshal(msg)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// readSSEFrame reads an SSE frame u to and including the blank-line separator.
func readSSEFrame(br *bufio.Reader) ([]byte, error) {
	var frame []byte
	for {
		line, err := br.ReadBytes('\n')
		if err != nil {
			return frame, err
		}
		frame = append(frame, line...)
		// Blank line (just \n or \r\n) terminates the frame
		if len(line) == 1 && line[0] == '\n' || len(line) == 2 && line[0] == '\r' && line[1] == '\n' {
			return frame, nil
		}
	}
}

func parseUsage(frame []byte) (prompt, completion int, ok bool) {
	idx := bytesIndex(frame, []byte("data: "))
	if idx < 0 {
		return 0, 0, false
	}
	jsonStart := idx + 6
	if jsonStart >= len(frame) {
		return 0, 0, false
	}

	var envelope struct {
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	err := json.Unmarshal(frame[jsonStart:], &envelope)
	if err != nil || envelope.Usage == nil {
		return 0, 0, false
	}
	return envelope.Usage.PromptTokens, envelope.Usage.CompletionTokens, true
}

// isDone returns true if the frame is the `data: [DONE]` sentinel.
func isDone(frame []byte) bool {
	return bytesHasPrefix(frame, []byte("data: [DONE]"))
}

// classifyReadErr categorizes read errors by context state
func classifyReadErr(ctx context.Context) string {
	select {
	case <-ctx.Done():
		return "client_gone"
	default:
		return "error"
	}
}

// SetSSEHeaders sets the required headers for an SSE response and returns the flusher
func (p *Relay) SetSSEHeaders(w http.ResponseWriter) (http.Flusher, bool) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	f, ok := w.(http.Flusher)
	return f, ok
}

func bytesIndex(b, substr []byte) int {
	for i := 0; i <= len(b)-len(substr); i++ {
		if string(b[i:i+len(substr)]) == string(substr) {
			return i
		}
	}
	return -1
}

func bytesHasPrefix(b, prefix []byte) bool {
	if len(b) < len(prefix) {
		return false
	}
	for i, c := range prefix {
		if b[i] != c {
			return false
		}
	}
	return true
}
