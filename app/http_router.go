package app

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (v *vxdb) newHTTPRouter() *mux.Router {
	router := mux.NewRouter()

	router.Use(servedMiddleware)
	router.Use(prometheusMiddleware)

	router.Handle("/metrics", promhttp.Handler())

	apiRouter := router.PathPrefix("/api").Subrouter()

	if value, ok := os.LookupEnv("AUTH_API_BASIC_USERPASS"); ok && len(value) > 5 {
		auth := NewAuthBasic(value)
		apiRouter.Use(auth.Middleware)
	} else if value, ok := os.LookupEnv("AUTH_API_JWKS_URL"); ok && len(value) > 5 {
		auth := AuthJWT{
			JwksURL: value,
		}

		apiRouter.Use(auth.Middleware)
	}

	if v.dbPerBucket {
		apiRouter.HandleFunc("/backup/{bucket}", v.apiBackup).Methods(http.MethodGet)
		apiRouter.HandleFunc("/restore/{bucket}", v.apiRestore).Methods(http.MethodPut)
	} else {
		apiRouter.HandleFunc("/backup", v.apiBackup).Methods(http.MethodGet)
		apiRouter.HandleFunc("/restore", v.apiRestore).Methods(http.MethodPut)
	}

	dataRouter := router.NewRoute().Subrouter()

	if value, ok := os.LookupEnv("AUTH_DATA_BASIC_USERPASS"); ok && len(value) > 5 {
		auth := NewAuthBasic(value)
		dataRouter.Use(auth.Middleware)
	} else if value, ok := os.LookupEnv("AUTH_DATA_JWKS_URL"); ok && len(value) > 5 {
		auth := AuthJWT{
			JwksURL: value,
		}

		dataRouter.Use(auth.Middleware)
	}

	if v.dbPerBucket {
		dataRouter.HandleFunc("/{bucket}", v.createBucket).Methods(http.MethodPut)
	}

	dataRouter.HandleFunc("/", v.listBuckets).Methods(http.MethodGet)
	dataRouter.HandleFunc("/{bucket}", v.listKeys).Methods(http.MethodGet)
	dataRouter.HandleFunc("/{bucket}", v.setKey).Methods(http.MethodPost)
	dataRouter.HandleFunc("/{bucket}/{key:.*}", v.getKey).Methods(http.MethodGet, http.MethodHead)
	dataRouter.HandleFunc("/{bucket}/{key:.*}", v.setKey).Methods(http.MethodPut)
	dataRouter.HandleFunc("/{bucket}/{key:.*}", v.delKey).Methods(http.MethodDelete)

	return router
}
