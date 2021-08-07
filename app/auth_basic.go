package app

import (
	"net/http"
	"strings"
)

type AuthBasic struct {
	users map[string]string
}

func NewAuthBasic(userpass string) *AuthBasic {
	mw := &AuthBasic{}
	mw.users = make(map[string]string)

	for _, userPassMap := range strings.Split(userpass, ";") {
		userPassSlice := strings.SplitN(userPassMap, ":", 2)
		mw.users[userPassSlice[0]] = userPassSlice[1]
	}

	return mw
}

func (mw *AuthBasic) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		pwd, ok2 := mw.users[user]

		if ok && ok2 && pass == pwd {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	})
}
