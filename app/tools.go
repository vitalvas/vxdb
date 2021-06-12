package app

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getBoolEnv(key string) bool {
	_, found := os.LookupEnv(key)
	return found
}

func getKeyByte(r *http.Request) []byte {
	vars := mux.Vars(r)

	return []byte(vars["key"])
}

func getHeaderKey(key string, r *http.Request) string {
	data := r.URL.Query().Get(key)
	if data != "" {
		return data
	}

	return r.Header.Get(fmt.Sprintf("x-key-%s", key))
}

func getNewKey() string {
	uid := uuid.Must(uuid.NewV4(), nil)

	return strings.ReplaceAll(uid.String(), "-", "")
}
