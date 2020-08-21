package keyring

import "github.com/99designs/keyring"

type Keyring interface {
	Set(keyring.Item) error
	Get(string) (keyring.Item, error)
	GetMetadata(string) (keyring.Metadata, error)
}
