package utils

import (
	"context"
	"errors"
	"net/http"
	"syscall"
)

func ClientGone(r *http.Request, err error) bool {
	return r.Context().Err() != nil ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET)
}
