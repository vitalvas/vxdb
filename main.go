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
)

func main() {
	dbOpts := badger.DefaultOptions(getEnv("DB_PATH", "./db"))
	dbOpts.ValueLogFileSize = 128 << 20

	db, err := badger.Open(dbOpts)
	if err != nil {
		log.Fatal(err)
	}

	vxdb := vxdb{
		db:         db,
		useBuckets: getBoolEnv("DB_USE_BUCKETS"),
	}

	defer vxdb.db.Close()

	router := mux.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("x-served-by", "VxDB")
			next.ServeHTTP(w, r)
		})
	})

	if vxdb.useBuckets {
		router.HandleFunc("/", vxdb.listBuckets).Methods("GET")
		router.HandleFunc("/{bucket}", vxdb.listKeys).Methods("GET")

		router.HandleFunc("/{bucket}", vxdb.setKey).Methods("POST")
		router.HandleFunc("/{bucket}/{key}", vxdb.getKey).Methods("GET", "HEAD")
		router.HandleFunc("/{bucket}/{key}", vxdb.setKey).Methods("PUT")
		router.HandleFunc("/{bucket}/{key}", vxdb.delKey).Methods("DELETE")

	} else {
		router.HandleFunc("/", vxdb.listKeys).Methods("GET")

		router.HandleFunc("/", vxdb.setKey).Methods("POST")
		router.HandleFunc("/{key}", vxdb.getKey).Methods("GET", "HEAD")
		router.HandleFunc("/{key}", vxdb.setKey).Methods("PUT")
		router.HandleFunc("/{key}", vxdb.delKey).Methods("DELETE")
	}

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
