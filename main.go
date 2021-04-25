package main

import (
	"log"
	"net/http"

	"github.com/dgraph-io/badger/v2"
	"github.com/gorilla/mux"
)

func main() {
	db, err := badger.Open(badger.DefaultOptions(getEnv("DB_PATH", "./db")))
	if err != nil {
		log.Fatal(err)
	}

	vxdb := vxdb{
		db:         db,
		useBuckets: getBoolEnv("DB_USE_BUCKETS"),
	}

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

	if err := srv.ListenAndServe(); err != nil {
		log.Println(err)
	}
}
