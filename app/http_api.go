package app

import (
	"net/http"

	"github.com/dgraph-io/badger/v4"
	"github.com/gorilla/mux"
)

func (v *vxdb) apiBackup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/octet-stream")
	w.Header().Set("content-disposition", "attachment; filename=backup.blob")

	var db *badger.DB

	if v.dbPerBucket {
		vars := mux.Vars(r)
		bucket := vars["bucket"]

		bdb, err := v.getDB(bucket)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		db = bdb

	} else {
		db = v.db
	}

	if _, err := db.Backup(w, 0); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (v *vxdb) apiRestore(w http.ResponseWriter, r *http.Request) {
	var db *badger.DB

	if v.dbPerBucket {
		vars := mux.Vars(r)
		bucket := vars["bucket"]

		bdb, err := v.getDB(bucket)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		db = bdb

	} else {
		db = v.db
	}

	if err := db.Load(r.Body, 256); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
