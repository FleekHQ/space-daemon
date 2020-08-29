package vault

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"golang.org/x/crypto/pbkdf2"
)

type vault struct {
	vaultAPIURL     string
	vaultSaltSecret string
}

type VaultItemType string

// Vault item types
const (
	PrivateKeyWithMnemonic VaultItemType = "PrivateKeyWithMnemonic"
)

type VkVersion string

const (
	VkVersion1 VkVersion = "V1"
)

// AES requires key length equal to 16, 24 or 32 bytes
const vaultKeyLength = 32

type VaultItem struct {
	ItemType VaultItemType
	Value    string
}

type storeVaultRequest struct {
	Vault string `json:"vault"`
	Vsk   string `json:"vsk"`
}

type StoredVault struct {
	Vault string
	Vsk   string
}

type retrieveVaultRequest struct {
	Vsk string `json:"vsk"`
}

type retrieveVaultResponse struct {
	EncryptedVault string `json:"encryptedVault"`
}

type Vault interface {
	Store(uuid string, passphrase string, apiToken string, items []VaultItem) (*StoredVault, error)
	Retrieve(uuid string, passphrase string) ([]VaultItem, error)
}

func New(vaultAPIURL string, vaultSaltSecret string) *vault {
	return &vault{
		vaultAPIURL:     vaultAPIURL,
		vaultSaltSecret: vaultSaltSecret,
	}
}

func (v *vault) Store(uuid string, passphrase string, apiToken string, items []VaultItem) (*StoredVault, error) {
	// Generate vault file
	vf, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}

	// Compute vault key
	vk := v.computeVk(uuid, passphrase, VkVersion1)

	// Encrypt vault file using vault key
	encVf, err := encrypt(vf, vk)
	if err != nil {
		return nil, err
	}

	// Compute vault service key
	vsk := v.computeVsk(vk, passphrase, VkVersion1)

	// Submit encrypted file and vsk to vault service
	storeRequest := &storeVaultRequest{
		Vault: base64.RawStdEncoding.EncodeToString(encVf),
		Vsk:   base64.RawStdEncoding.EncodeToString(vsk),
	}
	reqJSON, err := json.Marshal(storeRequest)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		CheckRedirect: http.DefaultClient.CheckRedirect,
	}
	req, err := http.NewRequest("POST", v.vaultAPIURL+"/vaults", bytes.NewBuffer(reqJSON))
	req.Header.Add("Authorization", apiToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	_, err = parseAPIResponse(resp)
	if err != nil {
		return nil, err
	}

	result := &StoredVault{
		Vault: storeRequest.Vault,
		Vsk:   storeRequest.Vsk,
	}
	return result, nil
}

func (v *vault) Retrieve(uuid string, passphrase string) ([]VaultItem, error) {
	// Compute vault key
	vk := v.computeVk(uuid, passphrase, VkVersion1)

	// Compute vault service key
	vsk := v.computeVsk(vk, passphrase, VkVersion1)

	// Send retrieve request to vault service
	reqJSON, err := json.Marshal(&retrieveVaultRequest{
		Vsk: base64.RawStdEncoding.EncodeToString(vsk),
	})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		v.vaultAPIURL+"/vaults/"+uuid,
		"application/json",
		bytes.NewBuffer(reqJSON),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := parseAPIResponse(resp)
	if err != nil {
		return nil, err
	}

	var parsedBody retrieveVaultResponse
	err = json.Unmarshal(body, &parsedBody)
	if err != nil {
		return nil, err
	}

	// Decrypt encrypted vault file
	encVfBase64 := parsedBody.EncryptedVault
	encVf, err := base64.RawStdEncoding.DecodeString(encVfBase64)
	if err != nil {
		return nil, err
	}

	vf, err := decrypt(encVf, vk)
	if err != nil {
		return nil, err
	}

	var items []VaultItem
	err = json.Unmarshal(vf, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (v *vault) computeVk(uuid string, pass string, version VkVersion) []byte {
	// In the future, we can increase iterations by doing a switch and mapping
	// version to a given amount of iterations.

	// version = V1 defaults to 100.000 iterations
	iterations := 100000

	return pbkdf2.Key([]byte(pass), []byte(string(version)+v.vaultSaltSecret+uuid), iterations, vaultKeyLength, sha512.New)
}

func (v *vault) computeVsk(vk []byte, pass string, version VkVersion) []byte {
	iterations := 100000

	return pbkdf2.Key(vk, []byte(string(version)+v.vaultSaltSecret+pass), iterations, vaultKeyLength, sha512.New)
}

func encrypt(data []byte, key []byte) ([]byte, error) {
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

func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
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

func parseAPIResponse(resp *http.Response) ([]byte, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		var returnedErr domain.APIError
		err = json.Unmarshal(body, &returnedErr)
		if err != nil {
			return nil, err
		}

		if returnedErr.Message != "" {
			return nil, errors.New(returnedErr.Message)
		}

		return nil, errors.New("Unexpected API error")
	}

	return body, nil
}
