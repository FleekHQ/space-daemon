package vault_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FleekHQ/space-daemon/core/vault"
	"github.com/stretchr/testify/assert"
)

const testSaltSecret = "someSecret"
const testUuid = "c907e7ef-7b36-4ab1-8a56-f788d7526a2c"
const testPassphrase = "banana1234"
const testAPIToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwdWJrZXkiOiJhZTRiMmFiNjU4ZmJiNzcyMjE0MDRkNjU3YzZiNzQyZDJlZjdjNTI2YjZhNWE5YzIwMGNjZjkzZmNhMWRjZTYzIiwidXVpZCI6ImM5MDdlN2VmLTdiMzYtNGFiMS04YTU2LWY3ODhkNzUyNmEyYyIsImlhdCI6MTU5ODI4NTA0MSwiZXhwIjoxNjAwODc3MDQxfQ.dgp8UhWCLjsU0SjxXwSb3g0jEurt2jAKPaY3B_eO-qE"

func TestVault_StoreAndRetrieve(t *testing.T) {
	testVaultItems := []vault.VaultItem{
		{
			ItemType: vault.PrivateKeyWithMnemonic,
			Value:    "SomePrivateKey",
		},
	}

	storeVaultMock := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}

	testBackupType := "password"

	serverMock := func() *httptest.Server {
		handler := http.NewServeMux()
		handler.HandleFunc("/vaults", storeVaultMock)

		srv := httptest.NewServer(handler)

		return srv
	}
	server := serverMock()
	defer server.Close()

	v := vault.New(
		// "https://f4nmmmkstb.execute-api.us-west-2.amazonaws.com/dev", // UNCOMMENT TO TEST REAL SERVER
		server.URL,
		testSaltSecret,
	)

	storeRequest, err := v.Store(testUuid, testPassphrase, testBackupType, testAPIToken, testVaultItems)

	assert.Nil(t, err)
	if err != nil {
		return
	}
	assert.NotNil(t, storeRequest)
	assert.NotNil(t, storeRequest.Vault)
	assert.NotEqual(t, "", storeRequest.Vault)
	assert.NotNil(t, storeRequest.Vsk)
	assert.NotEqual(t, "", storeRequest.Vsk)

	retrieveVaultMock := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"encryptedVault": "` + storeRequest.Vault + `"}`))
	}

	serverMock2 := func() *httptest.Server {
		handler := http.NewServeMux()
		handler.HandleFunc("/vaults/"+testUuid, retrieveVaultMock)

		srv := httptest.NewServer(handler)

		return srv
	}
	server2 := serverMock2()
	defer server2.Close()

	v2 := vault.New(
		// "https://f4nmmmkstb.execute-api.us-west-2.amazonaws.com/dev", // UNCOMMENT TO TEST REAL SERVER
		server2.URL,
		testSaltSecret,
	)

	retrievedItems, err := v2.Retrieve(testUuid, testPassphrase)
	assert.Nil(t, err)
	if err != nil {
		return
	}
	assert.NotNil(t, retrievedItems)

	// Assert response matches what we initially vaulted
	assert.Equal(t, testVaultItems[0].ItemType, retrievedItems[0].ItemType)
	assert.Equal(t, testVaultItems[0].Value, retrievedItems[0].Value)
}

func TestVault_StoreServerError(t *testing.T) {
	testVaultItems := []vault.VaultItem{
		{
			ItemType: vault.PrivateKeyWithMnemonic,
			Value:    "SomePrivateKey",
		},
	}

	testBackupType := "password"

	storeVaultMock := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{ "message": "Unauthorized Error: Authorization token is invalid."}`))
	}

	serverMock := func() *httptest.Server {
		handler := http.NewServeMux()
		handler.HandleFunc("/vaults", storeVaultMock)

		srv := httptest.NewServer(handler)

		return srv
	}
	server := serverMock()
	defer server.Close()

	v := vault.New(
		// "https://f4nmmmkstb.execute-api.us-west-2.amazonaws.com/dev", // UNCOMMENT TO TEST REAL SERVER
		server.URL,
		testSaltSecret,
	)

	storeRequest, err := v.Store(testUuid, testPassphrase, testBackupType, testAPIToken, testVaultItems)

	assert.NotNil(t, err)
	assert.Nil(t, storeRequest)
}

func TestVault_RetrieveServerError(t *testing.T) {
	retrieveVaultMock := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{ "message": "Unauthorized Error: Incorrect uuid or password."}`))
	}

	serverMock := func() *httptest.Server {
		handler := http.NewServeMux()
		handler.HandleFunc("/vaults/"+testUuid, retrieveVaultMock)

		srv := httptest.NewServer(handler)

		return srv
	}
	server := serverMock()
	defer server.Close()

	v := vault.New(
		// "https://f4nmmmkstb.execute-api.us-west-2.amazonaws.com/dev", // UNCOMMENT TO TEST REAL SERVER
		server.URL,
		testSaltSecret,
	)

	retrievedItems, err := v.Retrieve(testUuid, testPassphrase)

	assert.NotNil(t, err)
	assert.Nil(t, retrievedItems)
}
