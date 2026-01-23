package middleware

import (
	"net/http"

	"yadro.com/course/api/core"
)

func Rate(next http.HandlerFunc, rl core.RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rl.Submit(func() {
			next.ServeHTTP(w, r)
		})
	}
}
