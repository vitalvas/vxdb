package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dgraph-io/badger/v3"
)

func Execute(version, commit, date string) {
	log.Printf("Starting VxDBX %s+%s\n", version, commit)
	vxdbVersion.WithLabelValues(version, commit, date).Set(1)

	vxdb := vxdb{
		baseTableSize: 8 << 20, // 8MB
		dbPerBucket:   getBoolEnv("DB_PER_BUCKET"),
		dbPath:        getEnv("DB_PATH", "/var/lib/vxdb"),
	}

	defer vxdb.Close()

	if vxdb.dbPerBucket {
		vxdb.dbBucket = make(map[string]*badger.DB)
		vxdb.openDBBuckets()

	} else {
		if err := vxdb.Open("none"); err != nil {
			log.Fatal(err)
		}
	}

	go vxdb.runGC()

	srv := http.Server{
		Addr:    getEnv("HTTP_HOST", "0.0.0.0:8080"),
		Handler: vxdb.newHttpRouter(),
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
