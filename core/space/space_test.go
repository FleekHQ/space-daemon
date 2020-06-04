package space

import (
	"context"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/space/services"
	"github.com/FleekHQ/space-poc/core/textile/client"
	"github.com/FleekHQ/space-poc/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	cfg           *mocks.Config
	st            *mocks.Store
	textileClient *mocks.TextileClient
)

type TearDown func()

type GetTestDir func() string

func closeAndDelete(f *os.File) {
	f.Close()
	os.Remove(f.Name())
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


	getTestDir := func() string {
		return dir
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
	sv, getDir, tearDown := initTestService(t)
	defer tearDown()

	// setup mocks
	cfg.On("GetString", config.SpaceFolderPath, "").Return(
		getDir(),
		nil,
	)

	res, err := sv.ListDir(context.Background())

	assert.Nil(t, err)
	assert.NotEmpty(t, res)
	assert.Len(t, res, 3)
	if res[0].IsDir {
		// check for dir
		assert.True(t, res[0].IsDir)
		assert.Equal(t, getDir(), res[0].Path)
		assert.Equal(t, filepath.Base(getDir()), res[0].Name)
		assert.Equal(t, "", res[0].FileExtension)
	}

	assert.False(t, res[1].IsDir)
	assert.Equal(t, getDir()+"/test1.txt", res[1].Path)
	assert.Equal(t, "test1.txt", res[1].Name)
	assert.Equal(t, "txt", res[1].FileExtension)

	assert.False(t, res[2].IsDir)
	assert.Equal(t, getDir()+"/test2.pdf", res[2].Path)
	assert.Equal(t, "test2.pdf", res[2].Name)
	assert.Equal(t, "pdf", res[2].FileExtension)
}

// NOTE: update this test when it supports multiple buckets
func TestService_OpenFile(t *testing.T) {
	sv, getDir, tearDown := initTestService(t)
	defer tearDown()

	testKey := "bucketKey"
	testPath := "test.txt"

	// setup mocks
	cfg.On("GetString", config.SpaceFolderPath, "").Return(
		getDir(),
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
	assert.Contains(t, res.Location, getDir())
	assert.True(t, strings.HasSuffix(res.Location, testPath))

}
