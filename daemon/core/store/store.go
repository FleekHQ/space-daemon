package store

import (
	"errors"
	"fmt"
	"os"
	s "strings"

	"github.com/FleekHQ/space/daemon/log"

	badger "github.com/dgraph-io/badger"
	homedir "github.com/mitchellh/go-homedir"
)

const DefaultRootDir = "~/.fleek-space"
const BadgerFileName = "db"

type store struct {
	rootDir string
	db      *badger.DB
	isOpen  bool
}

type Store interface {
	Open() error
	Close() error
	Set(key []byte, value []byte) error
	SetString(key string, value string) error
	Get(key []byte) ([]byte, error)
	IsOpen() bool
}

type storeOptions struct {
	rootDir string
}

var defaultStoreOptions = storeOptions{
	rootDir: DefaultRootDir,
}

// Idea taken from here https://medium.com/soon-london/variadic-configuration-functions-in-go-8cef1c97ce99

type Option func(o *storeOptions)

func New(opts ...Option) Store {
	o := defaultStoreOptions
	for _, opt := range opts {
		opt(&o)
	}

	log.Info(fmt.Sprintf("using path %s for store", o.rootDir))

	store := &store{
		rootDir: o.rootDir,
		isOpen:  false,
	}

	return store
}

func (store *store) Open() error {
	if store.isOpen {
		return errors.New("Tried to open already open database")
	}

	rootDir := s.Join([]string{store.rootDir, BadgerFileName}, "/")

	if home, err := homedir.Dir(); err == nil {
		// If the root directory contains ~, we replace it with the actual home directory
		rootDir = s.Replace(rootDir, "~", home, -1)
	}

	// We create the directory in case it doesn't exist yet
	os.MkdirAll(rootDir, os.ModePerm)
	if db, err := badger.Open(badger.DefaultOptions(rootDir)); err != nil {
		return err
	} else {
		store.db = db
		store.isOpen = true
	}

	return nil
}

func (store store) IsOpen() bool {
	return store.isOpen
}

func (store *store) Close() error {
	if store.isOpen == false {
		return errors.New("Tried to close a not yet opened database")
	}

	defer store.db.Close()

	store.isOpen = false

	return nil
}

// Testing that store is correctly working
func (store *store) hotInit() {
	if err := store.Set([]byte("A"), []byte("B")); err != nil {
		log.Error("error", err)
		return
	}

	if val, err := store.Get([]byte("A")); err != nil {
		log.Error("error", err)
	} else {
		log.Info("Got store response")
		log.Info(string(val))
	}
}

// Helper function for setting store path
func WithPath(path string) Option {
	return func(o *storeOptions) {
		if path != "" {
			o.rootDir = path
		}
	}
}

func (store *store) getDb() (*badger.DB, error) {
	if store.isOpen == false {
		return nil, errors.New("Database has not been opened yet")
	}

	return store.db, nil
}

// Stores a key/value pair in the db.
func (store *store) Set(key []byte, value []byte) error {
	db, err := store.getDb()

	if err != nil {
		return err
	}

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

func (store *store) SetString(key string, value string) error {
	return store.Set([]byte(key), []byte(value))
}

// Given a key, retrieves the stored value. If the key is not found returns ErrKeyNotFound.
func (store *store) Get(key []byte) ([]byte, error) {
	db, err := store.getDb()

	if err != nil {
		return nil, err
	}

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
