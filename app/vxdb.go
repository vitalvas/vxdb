package app

import (
	"log"
	"time"

	"github.com/dgraph-io/badger/v3"
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
