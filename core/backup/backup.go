package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
)

type Backup struct {
	PrivateKey          string `json:"privateKey"`
	TextileClientBackup string `json:"textileClientBackup"`
}

// Note: Using static key since the goal of this is to obfuscate the file, not to encrypt it
var key = []byte{0xBC, 0xBC, 0xBC, 0xBC, 0xBC, 0xBC, 0xBC, 0xBC, 0xBC, 0xBC, 0xBC, 0xBC, 0xBC, 0xBC, 0xBC, 0xBC}

func obfuscate(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func deobfuscate(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("malformed ciphertext")
	}

	return gcm.Open(nil,
		ciphertext[:gcm.NonceSize()],
		ciphertext[gcm.NonceSize():],
		nil,
	)
}

// Creates a backup file in the given path
func MarshalBackup(path string, b *Backup) error {
	jsonData, err := json.Marshal(b)
	if err != nil {
		return err
	}

	obfuscatedBackup, err := obfuscate(jsonData)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, obfuscatedBackup, 0644)
	return err
}

// Reads a file in the given path and returns a Backup object
func UnmarshalBackup(path string) (*Backup, error) {
	obfuscatedBackup, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	jsonData, err := deobfuscate(obfuscatedBackup)
	if err != nil {
		return nil, err
	}

	var result Backup
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
