package middleware

import (
	"net/http"

	"yadro.com/course/api/core"
)

func Concurrency(next http.HandlerFunc, cl core.ConcurrencyLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		done := make(chan bool)
		response := cl.Submit(func() {
			next.ServeHTTP(w, r)
			done <- true
		})
		if response == core.StatusRejected {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		<-done
	}
}
