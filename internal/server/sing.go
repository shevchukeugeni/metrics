package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"net/http"
)

type SigningResponseWriter struct {
	http.ResponseWriter
	key string
}

func (r *SigningResponseWriter) Write(b []byte) (int, error) {
	h := hmac.New(sha256.New, []byte(r.key))
	h.Write(b)
	r.Header().Set("HashSHA256", string(h.Sum(nil)))
	size, err := r.ResponseWriter.Write(b)
	return size, err
}

func (r *SigningResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
}
