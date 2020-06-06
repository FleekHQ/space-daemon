package space

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/space/services"
	"github.com/FleekHQ/space-poc/core/textile/client"
	"github.com/FleekHQ/space-poc/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	buckets_pb "github.com/textileio/textile/api/buckets/pb"
)

var (
	cfg           *mocks.Config
	st            *mocks.Store
	textileClient *mocks.TextileClient
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
	textileClient = new(mocks.TextileClient)
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

	// NOTE: if we need to test without the store open we must override on each test
	st.On("IsOpen").Return(true)

	sv, err := NewService(st, textileClient, cfg)
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

func TestService_ListDir(t *testing.T) {
	sv, _, tearDown := initTestService(t)
	defer tearDown()

	bucketPath := "/ipfs/bafybeian44ntmjjfjbqt4dlkq4fiuhfzcxfunzuuzhbb7xkrnsdjb2sjha"

	testKey := "bucketKey"
	mockBuckets := []*client.TextileBucketRoot{
		{
			Key:  testKey,
			Name: "Personal Bucket",
			Path: "",
		},
	}

	mockDirItems := &client.TextileDirEntries{
		Item: &buckets_pb.ListPathReply_Item{
			Items: []*buckets_pb.ListPathReply_Item{
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

	mockDirItemsSubfolder := &client.TextileDirEntries{
		Item: &buckets_pb.ListPathReply_Item{
			Items: []*buckets_pb.ListPathReply_Item{
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

	textileClient.On("ListBuckets").Return(mockBuckets, nil)
	textileClient.On(
		"ListDirectory",
		mock.Anything,
		testKey,
		"",
	).Return(mockDirItems, nil)

	textileClient.On(
		"ListDirectory",
		mock.Anything,
		testKey,
		"/somedir",
	).Return(mockDirItemsSubfolder, nil)

	res, err := sv.ListDir(context.Background())

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
	assert.Equal(t, bucketPath+"/somedir/example.txt", res[1].Path)

	assert.False(t, res[2].IsDir)
	assert.Equal(t, "example.txt", res[2].Name)
	assert.Equal(t, "txt", res[2].FileExtension)
	assert.Equal(t, "bafkreia4q63he72sgzrn64kpa2uu5it7utmqkdby6t3xck6umy77x7p2ae", res[2].IpfsHash)
	assert.Equal(t, bucketPath+"/example.txt", res[2].Path)

	// assert mocks
	cfg.AssertExpectations(t)
}

// NOTE: update this test when it supports multiple buckets
func TestService_OpenFile(t *testing.T) {
	sv, getDir, tearDown := initTestService(t)
	defer tearDown()

	testKey := "bucketKey"
	testPath := "test.txt"

	// setup mocks
	cfg.On("GetString", config.SpaceFolderPath, "").Return(
		getDir().dir,
		nil,
	)

	cfg.On("GetInt", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(
		-1,
		nil,
	)

	mockBuckets := []*client.TextileBucketRoot{
		{
			Key:  testKey,
			Name: "Personal Bucket",
			Path: "",
		},
	}

	textileClient.On("ListBuckets").Return(mockBuckets, nil)
	textileClient.On(
		"GetFile",
		mock.Anything,
		testKey,
		testPath,
		mock.Anything,
	).Return(nil)

	res, err := sv.OpenFile(context.Background(), testPath, "")

	assert.Nil(t, err)
	assert.NotEmpty(t, res)
	assert.FileExists(t, res.Location)
	assert.Contains(t, res.Location, getDir().dir)
	assert.True(t, strings.HasSuffix(res.Location, testPath))
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

	mockBuckets := []*client.TextileBucketRoot{
		{
			Key:  testKey,
			Name: "Personal Bucket",
			Path: "",
		},
	}

	textileClient.On("ListBuckets").Return(mockBuckets, nil)

	textileClient.On(
		"UploadFile",
		mock.Anything,
		testKey,
		mock.Anything,
		mock.Anything,
	).Return(nil, nil, nil)


	err := sv.AddItems(context.Background(), testSourcePaths, bucketPath)

	assert.Nil(t, err)
	// assert mocks
	textileClient.AssertExpectations(t)
}

func TestService_AddItems_Folder(t *testing.T) {
	sv, getTempDir, tearDown := initTestService(t)
	defer tearDown()

	// setup tests
	testKey := "bucketKey"
	bucketPath := "/tests"
	testSourcePaths := []string{getTempDir().dir}

	mockBuckets := []*client.TextileBucketRoot{
		{
			Key:  testKey,
			Name: "Personal Bucket",
			Path: "",
		},
	}

	textileClient.On("ListBuckets").Return(mockBuckets, nil)

	textileClient.On(
		"CreateDirectory",
		mock.Anything,
		testKey,
		mock.Anything,
	).Return(nil, nil, nil)

	textileClient.On(
		"UploadFile",
		mock.Anything,
		testKey,
		mock.Anything,
		mock.Anything,
	).Return(nil, nil, nil)


	err := sv.AddItems(context.Background(), testSourcePaths, bucketPath)

	assert.Nil(t, err)
	// assert mocks
	textileClient.AssertExpectations(t)
}

func TestService_AddItems_OnError(t *testing.T) {
	sv, getTempDir, tearDown := initTestService(t)
	defer tearDown()

	// setup tests
	testKey := "bucketKey"
	bucketPath := "/tests"
	testSourcePaths := getTempDir().fileNames

	mockBuckets := []*client.TextileBucketRoot{
		{
			Key:  testKey,
			Name: "Personal Bucket",
			Path: "",
		},
	}

	textileClient.On("ListBuckets").Return(mockBuckets, nil)

	bucketError := errors.New("bucket failed")

	textileClient.On(
		"UploadFile",
		mock.Anything,
		testKey,
		mock.Anything,
		mock.Anything,
	).Return(nil, nil, bucketError)


	err := sv.AddItems(context.Background(), testSourcePaths, bucketPath)

	assert.NotNil(t, err)
	assert.Equal(t, bucketError, err)
	// assert mocks
	textileClient.AssertExpectations(t)
}