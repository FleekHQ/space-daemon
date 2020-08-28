package space

import (
	"context"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FleekHQ/space-daemon/config"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/stretchr/testify/mock"

	"github.com/FleekHQ/space-daemon/core/space/services"
	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/FleekHQ/space-daemon/core/vault"
	"github.com/FleekHQ/space-daemon/mocks"
	"github.com/stretchr/testify/assert"
	buckets_pb "github.com/textileio/textile/api/buckets/pb"
)

var (
	cfg            *mocks.Config
	st             *mocks.Store
	textileClient  *mocks.Client
	mockPath       *mocks.Path
	mockBucket     *mocks.Bucket
	mockEnv        *mocks.SpaceEnv
	mockSync       *mocks.Syncer
	mockKeychain   *mocks.Keychain
	mockVault      *mocks.Vault
	mockHub        *mocks.HubAuth
	mockPubKey     crypto.PubKey
	mockPrivKey    crypto.PrivKey
	mockPubKeyHex  string
	mockPrivKeyHex string
)

type TearDown func()

type GetTestDir func() *testDir

func closeAndDelete(f *os.File) {
	f.Close()
	os.Remove(f.Name())
}

type testDir struct {
	dir       string
	fileNames []string
}

func initTestService(t *testing.T) (*services.Space, GetTestDir, TearDown) {
	st = new(mocks.Store)
	cfg = new(mocks.Config)
	textileClient = new(mocks.Client)
	mockPath = new(mocks.Path)
	mockBucket = new(mocks.Bucket)
	mockEnv = new(mocks.SpaceEnv)
	mockSync = new(mocks.Syncer)
	mockKeychain = new(mocks.Keychain)
	mockVault = new(mocks.Vault)
	mockHub = new(mocks.HubAuth)
	var dir string
	var err error
	if dir, err = ioutil.TempDir("", "space-test-folders"); err != nil {
		t.Fatalf("error creating temp dir for tests %s", err.Error())
	}

	log.Println("temp dir", dir)

	tmpFile1, err := os.Create(dir + "/test1.txt")
	if err != nil {
		t.Fatalf("error creating temp file for tests %s", err.Error())
	}
	tmpFile2, err := os.Create(dir + "/test2.pdf")
	if err != nil {
		t.Fatalf("error creating temp file for tests %s", err.Error())
	}

	tmpFiles := []string{tmpFile1.Name(), tmpFile2.Name()}

	getTestDir := func() *testDir {
		return &testDir{
			dir:       dir,
			fileNames: tmpFiles,
		}
	}

	tearDown := func() {
		closeAndDelete(tmpFile1)
		closeAndDelete(tmpFile2)
		os.RemoveAll(dir)
	}

	cfg.On("GetString", config.Ipfsaddr, mock.Anything).Return(
		"/ip4/127.0.0.1/tcp/5001",
	)

	mockPubKeyHex = "67730a6678566ead5911d71304854daddb1fe98a396551a4be01de65da01f3a9"
	mockPrivKeyHex = "dd55f8921f90fdf31c6ef9ad86bd90605602fd7d32dc8ea66ab72deb6a82821c67730a6678566ead5911d71304854daddb1fe98a396551a4be01de65da01f3a9"

	pubKeyBytes, _ := hex.DecodeString(mockPubKeyHex)
	privKeyBytes, _ := hex.DecodeString(mockPrivKeyHex)
	mockPubKey, _ = crypto.UnmarshalEd25519PublicKey(pubKeyBytes)
	mockPrivKey, _ = crypto.UnmarshalEd25519PrivateKey(privKeyBytes)

	// NOTE: if we need to test without the store open we must override on each test
	st.On("IsOpen").Return(true)

	sv, err := NewService(st, textileClient, mockSync, cfg, mockKeychain, mockVault, mockHub, WithEnv(mockEnv))
	if err != nil {
		t.Fatal(err)
	}
	return sv.(*services.Space), getTestDir, tearDown
}

func TestNewService(t *testing.T) {
	sv, _, tearDown := initTestService(t)
	defer tearDown()

	assert.NotNil(t, sv)
}

func TestService_CreateBucket(t *testing.T) {
	sv, _, tearDown := initTestService(t)
	defer tearDown()

	slug := "testbucketslug"
	key := "testkey"
	path := "testpath"
	d1 := int64(1593405100)
	d2 := int64(1593405100)

	mb := &bucket.BucketData{
		Key:       key,
		Name:      slug,
		Path:      path,
		CreatedAt: d1,
		UpdatedAt: d2,
	}

	textileClient.On("CreateBucket", mock.Anything, mock.Anything).Return(mockBucket, nil)

	mockBucket.On(
		"GetData",
		mock.Anything,
	).Return(*mb, nil)

	res, err := sv.CreateBucket(context.Background(), "slug")

	assert.Nil(t, err)
	assert.NotEmpty(t, res)
	assert.Equal(t, key, res.GetData().Key)
	assert.Equal(t, slug, res.GetData().Name)
	assert.Equal(t, path, res.GetData().Path)
	assert.Equal(t, d1, res.GetData().CreatedAt)
	assert.Equal(t, d2, res.GetData().UpdatedAt)
}

func TestService_ListDirs(t *testing.T) {
	sv, _, tearDown := initTestService(t)
	defer tearDown()

	bucketPath := "/ipfs/bafybeian44ntmjjfjbqt4dlkq4fiuhfzcxfunzuuzhbb7xkrnsdjb2sjha"

	mockDirItems := &bucket.DirEntries{
		Item: &buckets_pb.ListPathItem{
			Items: []*buckets_pb.ListPathItem{
				{
					Path:  bucketPath + "/.textileseed",
					Name:  ".textileseed",
					IsDir: false,
					Size:  16,
					Cid:   "bafkreia4q63he72sgzrn64kpa2uu5it7utmqkdby6t3xck6umy77x7p2a1",
				},
				{
					Path:  bucketPath + "/somedir",
					Name:  "somedir",
					IsDir: true,
					Size:  0,
					Cid:   "",
				},
				{
					Path:  bucketPath + "/example.txt",
					Name:  "example.txt",
					IsDir: false,
					Size:  16,
					Cid:   "bafkreia4q63he72sgzrn64kpa2uu5it7utmqkdby6t3xck6umy77x7p2ae",
				},
			},
		},
	}

	mockDirItemsSubfolder := &bucket.DirEntries{
		Item: &buckets_pb.ListPathItem{
			Items: []*buckets_pb.ListPathItem{
				{
					Path:  bucketPath + "/somedir/example.txt",
					Name:  "example.txt",
					IsDir: false,
					Size:  16,
					Cid:   "bafkreia4q63he72sgzrn64kpa2uu5it7utmqkdby6t3xck6umy77x7p2ae",
				},
			},
		},
	}

	textileClient.On("GetDefaultBucket", mock.Anything).Return(mockBucket, nil)
	mockBucket.On(
		"ListDirectory",
		mock.Anything,
		"",
	).Return(mockDirItems, nil)

	mockBucket.On(
		"ListDirectory",
		mock.Anything,
		"/somedir",
	).Return(mockDirItemsSubfolder, nil)

	res, err := sv.ListDirs(context.Background(), "", "")

	assert.Nil(t, err)
	assert.NotEmpty(t, res)
	// .textileseed shouldn't be part of the reply
	assert.Len(t, res, 3)
	if res[0].IsDir {
		// check for dir
		assert.True(t, res[0].IsDir)
		assert.Equal(t, "", res[0].FileExtension)
	}

	assert.False(t, res[1].IsDir)
	assert.Equal(t, "example.txt", res[1].Name)
	assert.Equal(t, "txt", res[1].FileExtension)
	assert.Equal(t, "bafkreia4q63he72sgzrn64kpa2uu5it7utmqkdby6t3xck6umy77x7p2ae", res[1].IpfsHash)
	assert.Equal(t, "/somedir/example.txt", res[1].Path)

	assert.False(t, res[2].IsDir)
	assert.Equal(t, "example.txt", res[2].Name)
	assert.Equal(t, "txt", res[2].FileExtension)
	assert.Equal(t, "bafkreia4q63he72sgzrn64kpa2uu5it7utmqkdby6t3xck6umy77x7p2ae", res[2].IpfsHash)
	assert.Equal(t, "/example.txt", res[2].Path)

	// assert mocks
	cfg.AssertExpectations(t)
}

// NOTE: update this test when it supports multiple buckets
func TestService_OpenFile(t *testing.T) {
	sv, getDir, tearDown := initTestService(t)
	defer tearDown()

	testKey := "bucketKey"
	testPath := "/ipfs/bafybeievdakous3kamdgy6yxtmkvmibmro23kgf7xrduvwrxrlryzvu3sm/file.txt"
	testFileName := "file.txt"

	// setup mocks
	cfg.On("GetInt", mock.Anything, mock.Anything).Return(
		-1,
	)

	cfg.On("GetString", mock.Anything, mock.Anything).Return(
		"",
	)

	mockEnv.On("WorkingFolder").Return(
		getDir().dir,
	)

	mockSync.On("GetOpenFilePath", testKey, testPath).Return(
		"",
		false,
	)

	mockSync.On("AddFileWatch", mock.Anything).Return(
		nil,
	)

	textileClient.On("GetDefaultBucket", mock.Anything).Return(mockBucket, nil)
	mockBucket.On(
		"GetFile",
		mock.Anything,
		testPath,
		mock.Anything,
	).Return(nil)

	mockBucket.On(
		"Key",
	).Return(testKey)

	mockBucket.On(
		"Slug",
	).Return(testKey)

	res, err := sv.OpenFile(context.Background(), testPath, "")

	assert.Nil(t, err)
	assert.NotEmpty(t, res)
	assert.FileExists(t, res.Location)
	assert.Contains(t, res.Location, getDir().dir)
	assert.True(t, strings.HasSuffix(res.Location, testFileName))
	// assert mocks
	cfg.AssertExpectations(t)
	textileClient.AssertExpectations(t)
}

func TestService_AddItems_FilesOnly(t *testing.T) {
	sv, getTempDir, tearDown := initTestService(t)
	defer tearDown()

	// setup tests
	testKey := "bucketKey"
	bucketPath := "/tests"
	testSourcePaths := getTempDir().fileNames

	textileClient.On("GetDefaultBucket", mock.Anything).Return(mockBucket, nil)

	mockBucket.On(
		"Key",
	).Return(testKey)

	mockPath.On("String").Return("hash")

	for _, f := range testSourcePaths {
		_, fileName := filepath.Split(f)
		mockBucket.On(
			"UploadFile",
			mock.Anything,
			bucketPath+"/"+fileName,
			mock.Anything,
		).Return(nil, mockPath, nil)
	}

	ch, res, err := sv.AddItems(context.Background(), testSourcePaths, bucketPath, "")

	assert.Nil(t, err)
	assert.NotNil(t, ch)
	assert.NotEmpty(t, res)
	assert.Equal(t, int64(len(getTempDir().fileNames)), res.TotalFiles)

	count := 0
	for res := range ch {
		count++
		assert.NotNil(t, res)
		assert.Nil(t, res.Error)
		assert.NotEmpty(t, res.BucketPath)
		assert.NotEmpty(t, res.SourcePath)
	}

	assert.Equal(t, count, len(testSourcePaths))
	// assert mocks
	textileClient.AssertExpectations(t)
	mockBucket.AssertNumberOfCalls(t, "UploadFile", len(testSourcePaths))
}

func TestService_AddItems_Folder(t *testing.T) {
	sv, getTempDir, tearDown := initTestService(t)
	defer tearDown()

	// setup tests
	testKey := "bucketKey"
	bucketPath := "/tests"
	testSourcePaths := []string{getTempDir().dir}

	_, folderName := filepath.Split(getTempDir().dir)

	targetBucketPath := bucketPath + "/" + folderName

	textileClient.On("GetDefaultBucket", mock.Anything).Return(mockBucket, nil)

	mockBucket.On(
		"Key",
	).Return(testKey)

	mockPath.On("String").Return("hash")

	mockBucket.On(
		"CreateDirectory",
		mock.Anything,
		targetBucketPath,
	).Return(nil, mockPath, nil)

	for _, f := range getTempDir().fileNames {
		_, fileName := filepath.Split(f)
		mockBucket.On(
			"UploadFile",
			mock.Anything,
			targetBucketPath+"/"+fileName,
			mock.Anything,
		).Return(nil, mockPath, nil)
	}

	ch, res, err := sv.AddItems(context.Background(), testSourcePaths, bucketPath, "")

	assert.Nil(t, err)
	assert.NotNil(t, ch)
	assert.NotEmpty(t, res)
	assert.Equal(t, int64(len(getTempDir().fileNames)+1), res.TotalFiles)

	count := 0
	for res := range ch {
		count++
		assert.NotNil(t, res)
		assert.Nil(t, res.Error)
		assert.NotEmpty(t, res.BucketPath)
		assert.NotEmpty(t, res.SourcePath)
	}

	assert.Equal(t, count, len(testSourcePaths)+len(getTempDir().fileNames))
	// assert mocks
	textileClient.AssertExpectations(t)
	mockBucket.AssertNumberOfCalls(t, "UploadFile", len(getTempDir().fileNames))
	mockBucket.AssertNumberOfCalls(t, "CreateDirectory", 1)
}

func TestService_AddItems_OnError(t *testing.T) {
	sv, getTempDir, tearDown := initTestService(t)
	defer tearDown()

	// setup tests
	testKey := "bucketKey"
	bucketPath := "/tests"
	testSourcePaths := getTempDir().fileNames

	textileClient.On("GetDefaultBucket", mock.Anything).Return(mockBucket, nil)

	mockBucket.On(
		"Key",
	).Return(testKey)

	mockPath.On("String").Return("hash")

	bucketError := errors.New("bucket failed")

	mockBucket.On(
		"UploadFile",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(nil, nil, bucketError)

	ch, _, err := sv.AddItems(context.Background(), testSourcePaths, bucketPath, "")

	assert.Nil(t, err)
	assert.NotNil(t, ch)

	count := 0
	for res := range ch {
		count++
		assert.NotNil(t, res)
		assert.NotNil(t, res.Error)
		assert.NotEmpty(t, res.SourcePath)
		assert.Empty(t, res.BucketPath)
	}

	assert.Equal(t, count, len(testSourcePaths))
	// assert mocks
	textileClient.AssertExpectations(t)
	mockBucket.AssertNumberOfCalls(t, "UploadFile", len(getTempDir().fileNames))
}

func TestService_CreateIdentity(t *testing.T) {
	sv, _, tearDown := initTestService(t)
	defer tearDown()

	createIdentityMock := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{ "address": "0xd606f05a2a980f58737aa913553c8d6eac8b", "username": "dmerrill", "publicKey": "67730a6678566ead5911d71304854daddb1fe98a396551a4be01de65da01f3a9"}`))
	}

	serverMock := func() *httptest.Server {
		handler := http.NewServeMux()
		handler.HandleFunc("/identities", createIdentityMock)

		srv := httptest.NewServer(handler)

		return srv
	}

	server := serverMock()
	defer server.Close()
	cfg.On("GetString", mock.Anything, mock.Anything).Return(
		// "https://td4uiovozc.execute-api.us-west-2.amazonaws.com/dev", // UNCOMMENT TO TEST REAL SERVER
		server.URL,
	)

	testUsername := "dmerrill"

	mockKeychain.On(
		"GetStoredPublicKey",
	).Return(mockPubKey, nil)

	identity, err := sv.CreateIdentity(context.Background(), testUsername)

	assert.Nil(t, err)
	assert.NotNil(t, identity)
	assert.Equal(t, identity.PublicKey, mockPubKeyHex)
	assert.Equal(t, identity.Username, testUsername)
}

func TestService_CreateIdentity_OnError(t *testing.T) {
	sv, _, tearDown := initTestService(t)
	defer tearDown()

	createIdentityMock := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{ "message": "Validation Error: An identity with the given username already exists"}`))
	}

	serverMock := func() *httptest.Server {
		handler := http.NewServeMux()
		handler.HandleFunc("/identities", createIdentityMock)

		srv := httptest.NewServer(handler)

		return srv
	}

	server := serverMock()
	defer server.Close()
	cfg.On("GetString", mock.Anything, mock.Anything).Return(
		// "https://td4uiovozc.execute-api.us-west-2.amazonaws.com/dev", // UNCOMMENT TO TEST REAL SERVER
		server.URL,
	)

	testUsername := "dmerrill"

	mockKeychain.On(
		"GetStoredPublicKey",
	).Return(mockPubKey, nil)

	identity, err := sv.CreateIdentity(context.Background(), testUsername)

	assert.Nil(t, identity)
	assert.NotNil(t, err)
	assert.Equal(t, err, errors.New("Validation Error: An identity with the given username already exists"))
}

func TestService_GetIdentityByUsername(t *testing.T) {
	sv, _, tearDown := initTestService(t)
	defer tearDown()

	createIdentityMock := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{ "address": "0xd606f05a2a980f58737aa913553c8d6eac8b", "username": "dmerrill", "publicKey": "67730a6678566ead5911d71304854daddb1fe98a396551a4be01de65da01f3a9"}`))
	}

	serverMock := func() *httptest.Server {
		handler := http.NewServeMux()
		handler.HandleFunc("/identities/username/dmerrill", createIdentityMock)

		srv := httptest.NewServer(handler)

		return srv
	}

	server := serverMock()
	defer server.Close()
	cfg.On("GetString", mock.Anything, mock.Anything).Return(
		// "https://td4uiovozc.execute-api.us-west-2.amazonaws.com/dev", // UNCOMMENT TO TEST REAL SERVER
		server.URL,
	)

	testUsername := "dmerrill"

	identity, err := sv.GetIdentityByUsername(context.Background(), testUsername)

	assert.Nil(t, err)
	assert.NotNil(t, identity)
	assert.NotNil(t, identity.Address)
	assert.NotNil(t, identity.PublicKey)
	assert.Equal(t, identity.Username, testUsername)
}

func TestService_GetIdentityByUsername_OnError(t *testing.T) {
	sv, _, tearDown := initTestService(t)
	defer tearDown()

	createIdentityMock := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{ "message": "Not Found Error: Identity with username dmerrill1 not found." }`))
	}

	serverMock := func() *httptest.Server {
		handler := http.NewServeMux()
		handler.HandleFunc("/identities/username/dmerrill1", createIdentityMock)

		srv := httptest.NewServer(handler)

		return srv
	}

	server := serverMock()
	defer server.Close()
	cfg.On("GetString", mock.Anything, mock.Anything).Return(
		// "https://td4uiovozc.execute-api.us-west-2.amazonaws.com/dev", // UNCOMMENT TO TEST REAL SERVER
		server.URL,
	)

	testUsername := "dmerrill1"

	identity, err := sv.GetIdentityByUsername(context.Background(), testUsername)

	assert.Nil(t, identity)
	assert.NotNil(t, err)
	assert.Equal(t, err, errors.New("Not Found Error: Identity with username dmerrill1 not found."))
}

func TestService_GetPublicKey(t *testing.T) {
	sv, _, tearDown := initTestService(t)
	defer tearDown()

	mockKeychain.On(
		"GetStoredPublicKey",
	).Return(mockPubKey, nil)

	pub, err := sv.GetPublicKey(context.Background())

	assert.Nil(t, err)
	assert.NotNil(t, pub)
	assert.Equal(t, pub, mockPubKeyHex)
}

func TestService_BackupAndRestore(t *testing.T) {
	sv, getTestDir, tearDown := initTestService(t)
	defer tearDown()

	testDir := getTestDir()

	mockKeychain.On(
		"GetStoredKeyPairInLibP2PFormat",
	).Return(mockPrivKey, mockPubKey, nil)

	ctx := context.Background()

	path := testDir.fileNames[0]

	err := sv.CreateLocalKeysBackup(ctx, path)

	backup, _ := ioutil.ReadFile(path)

	assert.Nil(t, err)
	assert.NotNil(t, backup)

	mockKeychain.On("ImportExistingKeyPair", mock.Anything, mock.Anything).Return(nil)

	err = sv.RecoverKeysByLocalBackup(ctx, path)

	assert.Nil(t, err)
	mockKeychain.AssertCalled(t, "ImportExistingKeyPair", mockPrivKey, "")
}

func TestService_VaultBackup(t *testing.T) {
	sv, _, tearDown := initTestService(t)
	defer tearDown()

	pass := "strawberry123"
	uuid := "c907e7ef-7b36-4ab1-8a56-f788d7526a2c"
	ctx := context.Background()
	mnemonic := "clog chalk blame black uncover frame before decide tuition maple crowd uncle"

	mockKeychain.On(
		"GetStoredKeyPairInLibP2PFormat",
	).Return(mockPrivKey, mockPubKey, nil)

	mockKeychain.On("GetStoredMnemonic").Return(mnemonic, nil)

	mockVault.On("Store", uuid, pass, mock.Anything, mock.Anything).Return(nil, nil)

	mockHub.On("GetTokensWithCache", mock.Anything).Return(&hub.AuthTokens{
		AppToken: "",
		HubToken: "",
		Key:      "",
		Msg:      "",
		Sig:      "",
	}, nil)

	err := sv.BackupKeysByPassphrase(ctx, uuid, pass)
	assert.Nil(t, err)
	mockVault.AssertCalled(t, "Store", uuid, pass, mock.Anything, mock.Anything)
}

func TestService_VaultRestore(t *testing.T) {
	sv, _, tearDown := initTestService(t)
	defer tearDown()

	pass := "strawberry123"
	uuid := "c907e7ef-7b36-4ab1-8a56-f788d7526a2c"
	ctx := context.Background()
	mnemonic := "clog chalk blame black uncover frame before decide tuition maple crowd uncle"

	mockItem := vault.VaultItem{
		ItemType: vault.PrivateKeyWithMnemonic,
		Value:    mockPrivKeyHex + "___" + mnemonic,
	}

	mockItems := []vault.VaultItem{mockItem}

	mockVault.On("Retrieve", uuid, pass).Return(mockItems, nil)

	mockKeychain.On("ImportExistingKeyPair", mock.Anything, mock.Anything).Return(nil)

	err := sv.RecoverKeysByPassphrase(ctx, uuid, pass)
	assert.Nil(t, err)
	mockKeychain.AssertCalled(t, "ImportExistingKeyPair", mockPrivKey, mnemonic)
}
