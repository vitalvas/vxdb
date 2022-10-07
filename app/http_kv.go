package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
)

func (v *vxdb) listBuckets(w http.ResponseWriter, r *http.Request) {
	var listOfKeys []string

	if !v.dbPerBucket {
		keys := make(map[string]bool)

		err := v.db.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchValues = false

			it := txn.NewIterator(opts)
			defer it.Close()

			prefix := []byte(getHeaderKey("prefix", r))

			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()
				key := item.Key()
				multiKeys := strings.SplitN(string(key), "/", 2)
				if len(multiKeys) == 2 {
					keys[multiKeys[0]] = true
				}
			}

			return nil
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for k := range keys {
			listOfKeys = append(listOfKeys, k)
		}

	} else {
		for name := range v.dbBucket {
			listOfKeys = append(listOfKeys, name)
		}
	}

	data, err := json.Marshal(listOfKeys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	if string(data) == "null" {
		w.Write([]byte("[]"))
	} else {
		w.Write(data)
	}

	w.Write([]byte("\n"))
}

func (v *vxdb) listKeys(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	var db *badger.DB
	var listOfKeys []string

	if v.dbPerBucket {
		bdb, err := v.getDB(bucket)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		db = bdb
	} else {
		db = v.db
	}

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false

		it := txn.NewIterator(opts)
		defer it.Close()

		var prefix []byte

		if !v.dbPerBucket {
			prefix = append(prefix, []byte(bucket+"/")...)
		}

		userPrefix := getHeaderKey("prefix", r)
		if userPrefix != "" {
			prefix = append(prefix, []byte(userPrefix)...)
		}

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.Key()
			keyStr := string(key)
			multiKeys := strings.SplitN(keyStr, "/", 2)
			listOfKeys = append(listOfKeys, multiKeys[1])
		}

		return nil
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(listOfKeys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	if string(data) == "null" {
		w.Write([]byte("[]"))
	} else {
		w.Write(data)
	}

	w.Write([]byte("\n"))
}

func (v *vxdb) getKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	var db *badger.DB

	var key []byte

	if v.dbPerBucket {
		bdb, err := v.getDB(bucket)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		db = bdb
	} else {
		key = append(key, []byte(bucket+"/")...)
		db = v.db
	}

	key = append(key, getKeyByte(r)...)

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			w.Write(val)
			return nil
		})
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			http.Error(w, badger.ErrKeyNotFound.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (v *vxdb) setKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	var db *badger.DB

	key := getKeyByte(r)

	newKey := ""

	if len(key) == 0 {
		newKey = getNewKey()
		key = []byte(newKey)
	}

	if v.dbPerBucket {
		bdb, err := v.getDB(bucket)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		db = bdb
	} else {
		if _, exists := reservedKeys[bucket]; exists {
			http.Error(w, fmt.Sprintf("'%s' is a reserved name", bucket), http.StatusForbidden)
			return
		}

		key = append([]byte(bucket+"/"), key...)
		db = v.db
	}

	r.Body = http.MaxBytesReader(w, r.Body, v.baseTableSize)

	err := db.Update(func(txn *badger.Txn) error {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}

		e := badger.NewEntry(key, b)

		ttlStr := getHeaderKey("ttl", r)

		if ttlStr != "" {
			ttl, err := strconv.ParseInt(ttlStr, 10, 32)
			if err != nil {
				return nil
			}

			e.WithTTL(time.Second * time.Duration(ttl))
		}

		return txn.SetEntry(e)
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if newKey != "" {
		w.Header().Set("Location", fmt.Sprintf("/%s/%s", bucket, newKey))
	}

	w.WriteHeader(http.StatusCreated)
}

func (v *vxdb) delKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	var key []byte

	var db *badger.DB

	if v.dbPerBucket {
		bdb, err := v.getDB(bucket)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		db = bdb
	} else {
		key = append(key, []byte(bucket+"/")...)
		db = v.db
	}

	key = append(key, getKeyByte(r)...)

	err := db.Update(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		if err != nil {
			return err
		}
		return txn.Delete(key)
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			http.Error(w, badger.ErrKeyNotFound.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (v *vxdb) createBucket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	if _, exists := v.dbBucket[bucket]; exists {
		http.Error(w, "bucket alredy exists", http.StatusCreated)
		return
	}

	if _, exists := reservedKeys[bucket]; exists {
		http.Error(w, fmt.Sprintf("'%s' is a reserved name", bucket), http.StatusForbidden)
		return
	}

	if err := v.Open(bucket); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusCreated)
}
