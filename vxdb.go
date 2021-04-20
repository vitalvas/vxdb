package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dgraph-io/badger/v2"
)

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

func (v *vxdb) listKeys(w http.ResponseWriter, r *http.Request) {
	var listOfKeys []string

	err := v.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false

		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(getHeaderKey("prefix", r))

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.Key()
			listOfKeys = append(listOfKeys, string(key))
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
	err := v.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(getKeyByte(r))
		if err != nil {
			return err
		}
		item.Value(func(val []byte) error {
			w.Write(val)
			return nil
		})
		return nil
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
	key := getKeyByte(r)

	newKey := ""

	if len(key) == 0 {
		newKey = getNewKey()
		key = []byte(newKey)
	}

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
		w.Header().Set("Location", fmt.Sprintf("/api/v1/data/%s", newKey))
	}

	w.WriteHeader(http.StatusCreated)
}

func (v *vxdb) delKey(w http.ResponseWriter, r *http.Request) {
	err := v.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(getKeyByte(r))
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
