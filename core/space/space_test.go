package space

import (
	"context"
	"github.com/FleekHQ/space-poc/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
)

var (
	testMap map[string]string
	cfg     *mockCfg
	st      *mockStore
)

type mockCfg struct {
	mock.Mock
	testPath string
}

type TearDown func()

func (m mockCfg) GetString(key string, defaultValue interface{}) string {
	args := m.Called(key, defaultValue)

	return args.String(0)
}

func (m mockCfg) GetInt(key string, defaultValue interface{}) int {
	panic("implement me")
}

type mockStore struct {
	mock.Mock
}

func (s mockStore) IsOpen() bool {
	return true
}

func (s mockStore) Open() error {
	panic("implement me")
}

func (s mockStore) Close() error {
	panic("implement me")
}

func (s mockStore) Set(key []byte, value []byte) error {
	panic("implement me")
}

func (s mockStore) SetString(key string, value string) error {
	panic("implement me")
}

func (s mockStore) Get(key []byte) ([]byte, error) {
	panic("implement me")
}

func closeAndDelete(f *os.File) {
	f.Close()
	os.Remove(f.Name())
}

func initTestService(t *testing.T) (Service, TearDown) {
	st = new(mockStore)
	cfg = new(mockCfg)
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

	testMap = make(map[string]string)
	testMap[config.SpaceFolderPath] = dir

	tearDown := func() {
		closeAndDelete(tmpFile1)
		closeAndDelete(tmpFile2)
		os.RemoveAll(dir)
	}
	sv, err := NewService(st, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return sv, tearDown
}

func TestNewService(t *testing.T) {
	sv, tearDown := initTestService(t)
	defer tearDown()

	assert.NotNil(t, sv)
}

func TestService_ListDir(t *testing.T) {
	sv, tearDown := initTestService(t)
	defer tearDown()

	// setup mocks
	cfg.On("GetString", config.SpaceFolderPath, "").Return(
		testMap[config.SpaceFolderPath],
		nil,
	)

	res, err := sv.ListDir(context.Background())

	assert.Nil(t, err)
	assert.NotEmpty(t, res)
	assert.Len(t, res, 3)
	if res[0].IsDir {
		// check for dir
		assert.True(t, res[0].IsDir)
		assert.Equal(t, testMap[config.SpaceFolderPath], res[0].Path)
		assert.Equal(t, filepath.Base(testMap[config.SpaceFolderPath]), res[0].Name)
		assert.Equal(t, "", res[0].FileExtension)
	}

	assert.False(t, res[1].IsDir)
	assert.Equal(t, testMap[config.SpaceFolderPath] + "/test1.txt", res[1].Path)
	assert.Equal(t, "test1.txt", res[1].Name)
	assert.Equal(t, "txt", res[1].FileExtension)

	assert.False(t, res[2].IsDir)
	assert.Equal(t, testMap[config.SpaceFolderPath] + "/test2.pdf", res[2].Path)
	assert.Equal(t, "test2.pdf", res[2].Name)
	assert.Equal(t, "pdf", res[2].FileExtension)

}
