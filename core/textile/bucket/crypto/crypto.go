package crypto

import (
	"bytes"
	b64 "encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"strings"

	"github.com/FleekHQ/space-daemon/log"
)

func parseKeys(key []byte) (aesKey, iv, hmacKey []byte, err error) {
	if len(key) != aesKeySize+ivKeySize+hmacKeySize {
		return nil, nil, nil, errors.New("unsupported encryption keys provided.")
	}

	return key[:aesKeySize], key[aesKeySize:(aesKeySize + ivKeySize)], key[(aesKeySize + ivKeySize):], nil
}

// EncryptPathItems returns an encrypted path and a Reader that reads the encrypted data from the
// plain reader passed into the function.
// Encrypted data is AES-CTR of data + AES-512 HMAC of encrypted data
//
// NOTE: key must be a 64 byte long key
// To decrypt the result of this function use the DecryptPathItems function
func EncryptPathItems(key []byte, path string, plainReader io.Reader) (string, io.Reader, error) {
	// split key into key and secret
	// use key and secret and IV

	// encrypt path
	aesKey, iv, hmacKey, err := parseKeys(key)
	if err != nil {
		return "", nil, err
	}

	encryptedPath := ""
	pathParts := strings.Split(path, "/")
	pathPartsLen := len(pathParts)
	for i, pathItem := range pathParts {
		if pathItem == "" {
			continue
		}
		encryptedPathReader, err := NewEncryptReader(
			strings.NewReader(pathItem),
			aesKey,
			iv,
			hmacKey,
		)
		if err != nil {
			return "", nil, err
		}

		encryptedPathItem, err := readAsBase64Strings(encryptedPathReader)
		if err != nil {
			return "", nil, err
		}
		encryptedPath += encryptedPathItem
		if i != pathPartsLen-1 {
			encryptedPath += "/"
		}
	}

	// encrypt data
	var encryptedReader io.Reader
	if plainReader != nil {
		var err error
		encryptedReader, err = NewEncryptReader(
			plainReader,
			aesKey,
			iv,
			hmacKey,
		)
		if err != nil {
			return "", nil, err
		}
	}

	return encryptedPath, encryptedReader, nil
}

// DecryptPathItems returns a decrypted path string and an io.Reader that reads the decrypted data from the
// encrypted reader passed into the function.
// To only decrypt a path, pass in an empty byte buffer as the reader and check only for the string result.
//
// NOTE: key must be a 64 byte long key
func DecryptPathItems(key []byte, path string, encryptedReader io.Reader) (string, io.ReadCloser, error) {
	// decrypt path
	log.Debug("Decrypting Path Items", "path:"+path)
	aesKey, iv, hmacKey, err := parseKeys(key)
	if err != nil {
		return "", nil, err
	}

	decryptedPath := ""
	pathParts := strings.Split(path, "/")
	pathPartsLen := len(pathParts)
	for i, pathItem := range pathParts {
		if pathItem == "" {
			continue
		}
		encryptedEntryNameBytes, err := bytesFromBase64Strings(pathItem)
		if err != nil {
			return "", nil, err
		}

		decryptedPathReader, err := NewDecryptReader(
			bytes.NewBuffer(encryptedEntryNameBytes),
			aesKey,
			iv,
			hmacKey,
		)
		if err != nil {
			return "", nil, err
		}

		decryptedPathItem, err := readBufferString(decryptedPathReader)
		if err != nil {
			return "", nil, err
		}

		decryptedPath += decryptedPathItem
		if i != pathPartsLen-1 {
			decryptedPath += "/"
		}
	}

	// decrypt data
	var decryptedReader io.ReadCloser
	if encryptedReader != nil {
		var err error
		decryptedReader, err = NewDecryptReader(
			encryptedReader,
			aesKey,
			iv,
			hmacKey,
		)
		if err != nil {
			return "", nil, err
		}
	}

	return decryptedPath, decryptedReader, nil
}

func readBufferString(buf io.Reader) (string, error) {
	builder := new(strings.Builder)
	_, err := io.Copy(builder, buf)
	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

func readAsBase64Strings(buf io.Reader) (string, error) {
	data, err := ioutil.ReadAll(buf)
	if err != nil {
		return "", err
	}

	encodedData := b64.URLEncoding.EncodeToString(data)
	return encodedData, nil
}

func bytesFromBase64Strings(data string) ([]byte, error) {
	decodedData, err := b64.URLEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	return decodedData, nil
}
