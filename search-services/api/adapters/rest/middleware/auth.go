package middleware

import (
	"errors"
	"net/http"
	"strings"

	"yadro.com/course/api/core"
)

type TokenVerifier interface {
	Verify(token string) error
}

func Auth(next http.HandlerFunc, verifier TokenVerifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authParam := r.Header.Get("Authorization")
		if authParam == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		const bearerPrefix = "Token "
		if !strings.HasPrefix(authParam, bearerPrefix) {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authParam, bearerPrefix)

		err := verifier.Verify(token)
		if errors.Is(err, core.ErrUnauthorized) {
			http.Error(w, "Authorization is not passed", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, "Authorization has gone wrong", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	}
}
