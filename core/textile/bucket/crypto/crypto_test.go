package crypto

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

var validKeysSize = 80

func Test_EncryptPathItems_Fails_For_InvalidKeys(t *testing.T) {
	assert := require.New(t)

	key := make([]byte, 64)
	_, _ = rand.Read(key)

	_, _, err := EncryptPathItems(key, "", bytes.NewBufferString(""))
	assert.Error(err, "Encrypt Path Item with wrong key should fail")
}

func Test_DecryptPathItems_Fails_For_InvalidKeys(t *testing.T) {
	assert := require.New(t)

	key := make([]byte, 64)
	_, _ = rand.Read(key)

	_, _, err := DecryptPathItems(key, "", bytes.NewBufferString(""))
	assert.Error(err, "Decrypt Path Item with wrong key should fail")
}

func Test_EncryptPathItems_Works_With_DecryptPathItems(t *testing.T) {
	assert := require.New(t)

	key := make([]byte, validKeysSize)
	_, _ = rand.Read(key)
	plainPath := "smiggle/was/here"
	plainData := "This is the original unencrypted data"

	encryptedPath, encryptedReader, err := EncryptPathItems(key, plainPath, bytes.NewBufferString(plainData))
	assert.NoError(err, "Error encrypting Path Items")
	assert.NotEqual(encryptedPath, plainPath, "Encrypted Path is equal to Plain Path")

	decryptedPath, decryptedReader, err := DecryptPathItems(key, encryptedPath, encryptedReader)
	assert.NoError(err, "Error decrypting Path Items")
	assert.Equal(plainPath, decryptedPath, "Plain Path is not equal to decrypted path")
	decryptedData, err := ioutil.ReadAll(decryptedReader)
	assert.NoError(err)
	assert.Equal(plainData, bytes.NewBuffer(decryptedData).String(), "Plain data is not equal to decrypted data")
}

func Test_EncryptPathItems_And_DecryptPathItems_Work_With_TopLevel_Files(t *testing.T) {
	assert := require.New(t)

	key := make([]byte, validKeysSize)
	_, _ = rand.Read(key)
	plainPath := "single_directory_entry"

	encryptedPath, _, err := EncryptPathItems(key, plainPath, nil)
	assert.NoError(err, "Error encrypting Path Items")
	assert.NotEqual(encryptedPath, plainPath, "Encrypted Path is equal to Plain Path")

	decryptedPath, _, err := DecryptPathItems(key, encryptedPath, nil)
	assert.NoError(err, "Error decrypting Path Items")
	assert.Equal(plainPath, decryptedPath, "Plain Path is not equal to decrypted path")
}
