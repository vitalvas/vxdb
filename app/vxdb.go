package app

import (
	"encoding/base64"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v4"
)

var reservedKeys = map[string]bool{
	"metrics": true,
	"admin":   true,
	"api":     true,
}

type vxdb struct {
	db            *badger.DB
	dbBucket      map[string]*badger.DB
	dbPath        string
	dbPerBucket   bool
	baseTableSize int64
}

func (v *vxdb) Open(name string) error {
	dbPath := v.dbPath

	if v.dbPerBucket {
		dbPath = filepath.Join(v.dbPath, name)
	}

	dbOpts := badger.DefaultOptions(dbPath)
	dbOpts = dbOpts.WithValueLogFileSize(128 << 20) // 128MB
	dbOpts = dbOpts.WithIndexCacheSize(128 << 20)   // 128MB
	dbOpts = dbOpts.WithBaseTableSize(v.baseTableSize)
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
		return err
	}

	if v.dbPerBucket {
		v.dbBucket[name] = db
	} else {
		v.db = db
	}

	return nil
}

func (v *vxdb) openDBBuckets() error {
	if v.dbBucket == nil {
		v.dbBucket = make(map[string]*badger.DB)
	}

	return filepath.Walk(v.dbPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if (info.IsDir() && path == v.dbPath) || !info.IsDir() {
			return nil
		}

		if _, exists := reservedKeys[info.Name()]; exists {
			return nil
		}

		return v.Open(info.Name())
	})
}

func (v *vxdb) getDB(bucket string) (*badger.DB, error) {
	if v.dbPerBucket {
		if db, ok := v.dbBucket[bucket]; ok {
			return db, nil
		}

		return nil, errors.New("bucket not found")
	}

	return v.db, nil
}

func (v *vxdb) Close() {
	if v.dbPerBucket {
		for _, db := range v.dbBucket {
			db.Close()
		}
	} else {
		v.db.Close()
	}
}

func (v *vxdb) runGC() {
	for {
		time.Sleep(10 * time.Minute)

		if v.dbPerBucket {
			for _, db := range v.dbBucket {
				if err := db.RunValueLogGC(0.7); err != nil {
					if err != badger.ErrNoRewrite {
						log.Fatal(err)
					}
				}
			}
		} else {
			if err := v.db.RunValueLogGC(0.7); err != nil {
				if err != badger.ErrNoRewrite {
					log.Fatal(err)
				}
			}
		}
	}
}
