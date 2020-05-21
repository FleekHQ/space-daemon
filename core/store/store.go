package store

import (
	"os"
	s "strings"

	badger "github.com/dgraph-io/badger/v2"
	homedir "github.com/mitchellh/go-homedir"
)

const DefaultRootDir = "~/.fleek-space"
const BadgerFileName = "db"

type Store struct {
	rootDir string
}

func New(appRootDir ...string) *Store {
	rootDir := DefaultRootDir
	if appRootDir != nil {
		rootDir = appRootDir[0]
	}

	return &Store{
		rootDir: rootDir,
	}
}

func (store *Store) getDb() (*badger.DB, error) {
	rootDir := s.Join([]string{store.rootDir, BadgerFileName}, "/")

	if home, err := homedir.Dir(); err == nil {
		// If the root directory contains ~, we replace it with the actual home directory
		rootDir = s.Replace(rootDir, "~", home, -1)
	}

	// We create the directory in case it doesn't exist yet
	os.MkdirAll(rootDir, os.ModePerm)
	db, err := badger.Open(badger.DefaultOptions(rootDir))

	if err != nil {
		// Could not open the local database file
		return nil, err
	}

	return db, nil
}

// Stores a key/value pair in the db.
func (store *Store) Set(key []byte, value []byte) error {
	db, err := store.getDb()

	if err != nil {
		return err
	}

	defer db.Close()

	updateHandler := func(txn *badger.Txn) error {
		e := badger.NewEntry(key, value)
		err := txn.SetEntry(e)
		return err
	}

	if err := db.Update(updateHandler); err != nil {
		return err
	}

	return nil
}

func (store *Store) SetString(key string, value string) error {
	return store.Set([]byte(key), []byte(value))
}

// Given a key, retrieves the stored value. If the key is not found returns ErrKeyNotFound.
func (store *Store) Get(key []byte) ([]byte, error) {
	db, err := store.getDb()

	if err != nil {
		return nil, err
	}

	defer db.Close()

	var valCopy []byte

	transactionHandler := func(txn *badger.Txn) error {
		if item, err := txn.Get(key); err != nil {
			return err
		} else {
			err = item.Value(func(val []byte) error {
				// Copying or parsing val is valid.
				valCopy = append([]byte{}, val...)

				return nil
			})

			if err != nil {
				return err
			}

			return nil
		}

	}

	if err = db.View(transactionHandler); err != nil {
		return nil, err
	}

	return valCopy, nil
}
