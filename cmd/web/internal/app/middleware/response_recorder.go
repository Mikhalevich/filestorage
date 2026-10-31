package middleware

import (
	"fmt"
	"net/http"
)

type responseRecorder struct {
	http.ResponseWriter

	status      int
	bytes       int
	wroteHeader bool
}

func (r *responseRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}

	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.wroteHeader = true

	written, err := r.ResponseWriter.Write(b)
	r.bytes += written

	if err != nil {
		return written, fmt.Errorf("write response: %w", err)
	}

	return written, nil
}

func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
