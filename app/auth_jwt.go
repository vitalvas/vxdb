package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/lestrrat-go/jwx/jwk"
)

type AuthJWT struct {
	JwksURL string
}

func (mw *AuthJWT) getKey(token *jwt.Token) (interface{}, error) {
	set, err := jwk.Fetch(context.Background(), mw.JwksURL)
	if err != nil {
		return nil, err
	}

	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("expecting JWT header to have string kid")
	}

	if key, ok := set.LookupKeyID(keyID); ok {
		var keyRaw interface{}
		if err := key.Raw(&keyRaw); err != nil {
			return nil, err
		}

		return keyRaw, nil
	}

	return nil, fmt.Errorf("unable to find key %q", keyID)
}

func (mw *AuthJWT) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := strings.SplitN(r.Header.Get("authorization"), " ", 2)

		if len(auth) != 2 || strings.ToLower(auth[0]) != "bearer" {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(auth[1], mw.getKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("authorization failed: %s", err), http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			http.Error(w, "authorization failed: token is invalid", http.StatusUnauthorized)
			return
		}

		claims := token.Claims.(jwt.MapClaims)

		if iss, ok := claims["iss"]; ok {
			w.Header().Set("x-auth-iss", iss.(string))
		}

		if jti, ok := claims["jti"]; ok {
			w.Header().Set("x-auth-jti", jti.(string))
		}

		next.ServeHTTP(w, r)
	})
}
