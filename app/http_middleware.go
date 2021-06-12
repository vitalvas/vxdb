package app

import "net/http"

func servedMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-served-by", "VxDB")
		next.ServeHTTP(w, r)
	})
}
