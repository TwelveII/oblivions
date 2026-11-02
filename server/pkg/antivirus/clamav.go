package antivirus

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type Status string

const (
	StatusClean       Status = "clean"
	StatusInfected    Status = "infected"
	StatusTooLarge    Status = "too_large"
	StatusUnavailable Status = "unavailable"
)

type Result struct {
	Status    Status
	Signature string
	Duration  time.Duration
	Err       error
}

type Scanner interface {
	Scan(ctx context.Context, path string) Result
}

type ClamAVScanner struct {
	addr    string
	timeout time.Duration
}

func NewClamAVScanner(addr string, timeout time.Duration) *ClamAVScanner {
	return &ClamAVScanner{addr: addr, timeout: timeout}
}

// parseClamAVAddr maps a config address to a (network, address) pair.
// "unix:/path" or a bare "/path" → unix socket; anything else → tcp.
func parseClamAVAddr(addr string) (network, address string) {
	if strings.HasPrefix(addr, "unix:") {
		return "unix", strings.TrimPrefix(addr, "unix:")
	}
	if strings.HasPrefix(addr, "/") {
		return "unix", addr
	}
	return "tcp", addr
}

func (s *ClamAVScanner) Scan(ctx context.Context, path string) Result {
	start := time.Now()
	if s.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	network, addr := parseClamAVAddr(s.addr)
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return Result{Status: StatusUnavailable, Duration: time.Since(start), Err: err}
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	f, err := openFile(path)
	if err != nil {
		return Result{Status: StatusUnavailable, Duration: time.Since(start), Err: err}
	}
	defer f.Close()

	if _, err := conn.Write([]byte("nINSTREAM\n")); err != nil {
		return Result{Status: StatusUnavailable, Duration: time.Since(start), Err: err}
	}

	buf := make([]byte, 1024*1024)
	lenBuf := make([]byte, 4)
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			lenBuf[0] = byte(n >> 24)
			lenBuf[1] = byte(n >> 16)
			lenBuf[2] = byte(n >> 8)
			lenBuf[3] = byte(n)
			if _, err := conn.Write(lenBuf); err != nil {
				return Result{Status: StatusUnavailable, Duration: time.Since(start), Err: err}
			}
			if _, err := conn.Write(buf[:n]); err != nil {
				return Result{Status: StatusUnavailable, Duration: time.Since(start), Err: err}
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return Result{Status: StatusUnavailable, Duration: time.Since(start), Err: readErr}
		}
	}

	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return Result{Status: StatusUnavailable, Duration: time.Since(start), Err: err}
	}

	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return Result{Status: StatusUnavailable, Duration: time.Since(start), Err: err}
	}
	return parseClamdResponse(line, time.Since(start))
}

func parseClamdResponse(line string, duration time.Duration) Result {
	line = strings.Trim(strings.TrimSpace(line), "\x00")
	lower := strings.ToLower(line)
	switch {
	case strings.HasSuffix(line, " OK"):
		return Result{Status: StatusClean, Duration: duration}
	case strings.Contains(line, " FOUND"):
		sig := line
		if idx := strings.LastIndex(line, ": "); idx >= 0 {
			sig = strings.TrimSuffix(line[idx+2:], " FOUND")
		}
		return Result{Status: StatusInfected, Signature: sig, Duration: duration}
	case strings.Contains(lower, "size limit exceeded"):
		return Result{Status: StatusTooLarge, Duration: duration, Err: fmt.Errorf("clamd stream size limit exceeded")}
	default:
		return Result{Status: StatusUnavailable, Duration: duration, Err: fmt.Errorf("unexpected clamd response: %s", line)}
	}
}
