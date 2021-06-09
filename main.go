package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	dbOpts := badger.DefaultOptions(getEnv("DB_PATH", "./db"))
	dbOpts.ValueLogFileSize = 128 << 20

	db, err := badger.Open(dbOpts)
	if err != nil {
		log.Fatal(err)
	}

	vxdb := vxdb{
		db: db,
	}

	defer vxdb.db.Close()

	router := mux.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("x-served-by", "VxDB")
			next.ServeHTTP(w, r)
		})
	})

	router.Use(prometheusMiddleware)
	router.Handle("/metrics", promhttp.Handler())

	router.HandleFunc("/api/backup", vxdb.apiBackup).Methods(http.MethodGet)
	router.HandleFunc("/api/restore", vxdb.apiRestore).Methods(http.MethodPut)

	router.HandleFunc("/", vxdb.listBuckets).Methods(http.MethodGet)
	router.HandleFunc("/{bucket}", vxdb.listKeys).Methods(http.MethodGet)
	router.HandleFunc("/{bucket}", vxdb.setKey).Methods(http.MethodPost)
	router.HandleFunc("/{bucket}/{key:.*}", vxdb.getKey).Methods(http.MethodGet, http.MethodHead)
	router.HandleFunc("/{bucket}/{key:.*}", vxdb.setKey).Methods(http.MethodPut)
	router.HandleFunc("/{bucket}/{key:.*}", vxdb.delKey).Methods(http.MethodDelete)

	srv := http.Server{
		Addr:    getEnv("HTTP_HOST", "0.0.0.0:8080"),
		Handler: router,
	}

	go vxdb.runGC()

	go func() {
		signChan := make(chan os.Signal, 1)

		signal.Notify(signChan, os.Interrupt, syscall.SIGTERM)
		sig := <-signChan
		log.Println("shutdown:", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("HTTP server shutdown failed:%+s", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalln(err)
	}
}
