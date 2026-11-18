package utils

import (
	"io"
	"net/http"
	"time"
)

type idleReader struct {
	r    io.Reader
	rc   *http.ResponseController
	idle time.Duration
	last time.Time
}

func NewIdleReader(w http.ResponseWriter, r io.Reader, idle time.Duration) io.Reader {
	ir := &idleReader{r: r, rc: http.NewResponseController(w), idle: idle}
	d := time.Now().Add(ir.idle)
	_ = ir.rc.SetWriteDeadline(d) // ErrNotSupported on HTTP/3, fine
	_ = ir.rc.SetReadDeadline(d)  // ErrNotSupported on HTTP/3, fine
	return ir
}

func (ir *idleReader) Read(p []byte) (int, error) {
	n, err := ir.r.Read(p)
	if n > 0 && time.Since(ir.last) > time.Second { // No need to reset timer every ms
		ir.last = time.Now()
		d := time.Now().Add(ir.idle)
		_ = ir.rc.SetWriteDeadline(d) // ErrNotSupported on HTTP/3, fine
		_ = ir.rc.SetReadDeadline(d)  // ErrNotSupported on HTTP/3, fine
	}
	return n, err
}
