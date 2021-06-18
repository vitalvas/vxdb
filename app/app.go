package app

import (
	"context"
	"encoding/base64"
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

func Execute(version, commit, date string) {
	log.Printf("Starting VxDBX %s+%s\n", version, commit)
	vxdbVersion.WithLabelValues(version, commit, date).Set(1)

	vxdb := vxdb{
		baseTableSize: 8 << 20, // 8MB
	}

	defaultDBPath := "/var/lib/vxdb"
	if version == "dev" {
		defaultDBPath = "./db"
	}

	dbOpts := badger.DefaultOptions(getEnv("DB_PATH", defaultDBPath))
	dbOpts = dbOpts.WithValueLogFileSize(128 << 20) // 128MB
	dbOpts = dbOpts.WithIndexCacheSize(128 << 20)   // 128MB
	dbOpts = dbOpts.WithBaseTableSize(vxdb.baseTableSize)
	dbOpts = dbOpts.WithCompactL0OnClose(true)

	if value, ok := os.LookupEnv("ENCRYPTION_KEY"); ok {
		data, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			log.Fatal(err)
		}

		if len(data) != 16 && len(data) != 24 && len(data) != 32 {
			log.Fatal("Encryption key's length should beeither 16, 24, or 32 bytes")
		}
		dbOpts = dbOpts.WithEncryptionKey(data)
		dbOpts = dbOpts.WithEncryptionKeyRotationDuration(7 * 24 * time.Hour) // 7 days
	}

	db, err := badger.Open(dbOpts)
	if err != nil {
		log.Fatal(err)
	}

	vxdb.db = db

	defer vxdb.db.Close()

	go vxdb.runGC()

	router := mux.NewRouter()

	router.Use(servedMiddleware)
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
