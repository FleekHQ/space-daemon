package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha512"
	"errors"
	"hash"
	"io"
)

const (
	aesKeySize  = 32
	ivKeySize   = 16
	hmacKeySize = 32
	hmacSize    = 64
)

var hashFunc = sha512.New

// hashReadWriter hashes on write and on read finalizes the hash and returns it.
// Writes after a Read will return an error.
type hashReadWriter struct {
	hash hash.Hash
	done bool
	sum  io.Reader
}

// Write implements io.Writer
func (h *hashReadWriter) Write(p []byte) (int, error) {
	if h.done {
		return 0, errors.New("writing to hashReadWriter after read is not allowed")
	}
	return h.hash.Write(p)
}

// Read implements io.Reader.
func (h *hashReadWriter) Read(p []byte) (int, error) {
	if !h.done {
		h.done = true
		h.sum = bytes.NewReader(h.hash.Sum(nil))
	}
	return h.sum.Read(p)
}

// NewEncryptReader returns an io.Reader wrapping the provided io.Reader.
func NewEncryptReader(r io.Reader, aesKey, iv, hmacKey []byte) (io.Reader, error) {
	if len(aesKey) != aesKeySize {
		return nil, errors.New("encryption key has incorrect length")
	}

	if len(iv) != ivKeySize {
		return nil, errors.New("encryption initialization vector size has incorrect length")
	}

	if len(hmacKey) != hmacKeySize {
		return nil, errors.New("encryption hmac key has incorrect length")
	}

	return newEncryptReader(r, aesKey, iv, hmacKey)
}

func newEncryptReader(r io.Reader, aesKey, iv, hmacKey []byte) (io.Reader, error) {
	b, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	h := hmac.New(hashFunc, hmacKey)
	hr := &hashReadWriter{hash: h}
	sr := &cipher.StreamReader{R: r, S: cipher.NewCTR(b, iv)}
	return io.MultiReader(io.TeeReader(sr, hr), hr), nil
}
