package space

import (
	"context"
	"errors"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FleekHQ/space-poc/core/space/services"
	"github.com/FleekHQ/space-poc/core/textile"
	"github.com/FleekHQ/space-poc/mocks"
	"github.com/stretchr/testify/assert"
	buckets_pb "github.com/textileio/textile/api/buckets/pb"
)

var (
	cfg           *mocks.Config
	st            *mocks.Store
	textileClient *mocks.Client
	mockPath      *mocks.Path
	mockBucket    *mocks.Bucket
	mockEnv       *mocks.SpaceEnv
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

	sv, err := NewService(st, textileClient, cfg, WithEnv(mockEnv))
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

	mockDirItems := &textile.DirEntries{
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

	mockDirItemsSubfolder := &textile.DirEntries{
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

	ch, err := sv.AddItems(context.Background(), testSourcePaths, bucketPath)

	assert.Nil(t, err)
	assert.NotNil(t, ch)

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

	ch, err := sv.AddItems(context.Background(), testSourcePaths, bucketPath)

	assert.Nil(t, err)
	assert.NotNil(t, ch)

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

	ch, err := sv.AddItems(context.Background(), testSourcePaths, bucketPath)

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
