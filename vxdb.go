package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
)

var reservedKeys = map[string]bool{
	"metrics": true,
	"admin":   true,
	"api":     true,
}

type vxdb struct {
	db *badger.DB
}

func (v *vxdb) runGC() {
	for {
		time.Sleep(time.Minute)
		if err := v.db.RunValueLogGC(0.7); err != nil {
			if err != badger.ErrNoRewrite {
				log.Fatal(err)
			}
		}
	}
}

func (v *vxdb) listBuckets(w http.ResponseWriter, r *http.Request) {
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

	var listOfKeys []string

	for k := range keys {
		listOfKeys = append(listOfKeys, k)
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

	var listOfKeys []string

	err := v.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false

		it := txn.NewIterator(opts)
		defer it.Close()

		var prefix []byte

		prefix = append(prefix, []byte(vars["bucket"]+"/")...)

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

	var key []byte

	key = append(key, []byte(bucket+"/")...)
	key = append(key, getKeyByte(r)...)

	err := v.db.View(func(txn *badger.Txn) error {
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

	key := getKeyByte(r)

	newKey := ""

	if len(key) == 0 {
		newKey = getNewKey()
		key = []byte(newKey)
	}

	if _, exists := reservedKeys[bucket]; exists {
		http.Error(w, fmt.Sprintf("'%s' is a reserved name", bucket), http.StatusForbidden)
		return
	}

	key = append([]byte(bucket+"/"), key...)

	err := v.db.Update(func(txn *badger.Txn) error {
		b, err := ioutil.ReadAll(r.Body)
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

	key = append(key, []byte(bucket+"/")...)
	key = append(key, getKeyByte(r)...)

	err := v.db.Update(func(txn *badger.Txn) error {
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
