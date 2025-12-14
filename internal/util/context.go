package util

import "net/http"

type ctxKey string

const (
	CtxRetryKey    ctxKey = "retry"
	CtxAttemptsKey ctxKey = "attempts"
)

func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(CtxRetryKey).(int); ok {
		return retry
	}
	return 0
}

func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(CtxAttemptsKey).(int); ok {
		return attempts
	}
	return 0
}
