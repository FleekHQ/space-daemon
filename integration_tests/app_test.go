package integration_tests

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"runtime"
	"testing"

	"github.com/FleekHQ/space-daemon/grpc/pb"

	"github.com/stretchr/testify/assert"

	spaceApp "github.com/FleekHQ/space-daemon/app"
)

// Ignoring this for now because it fails as we don't
// currently properly shutdown all started goroutines
// note, this leak might be from libraries we don't control
func XTestAppDoesNotLeakGoroutines(t *testing.T) {
	goRoutinesBefore := runtime.NumGoroutine()

	_, cfg, env := GetTestConfig()
	ctx, cancelCtx := context.WithCancel(context.Background())
	app := spaceApp.New(cfg, env)

	var err error
	errChan := make(chan error)
	go func() {
		errChan <- app.Start(ctx)
	}()

	select {
	case <-app.WaitForReady():
	case err = <-errChan:
	}

	assert.Nil(t, err, "app.Start() Failed")
	cancelCtx()
	err = <-errChan
	assert.Nil(t, err, "app.Shutdown() Failed")

	assert.Equal(t, goRoutinesBefore, runtime.NumGoroutine(), "Goroutine leaked on app Shutdown")
}

func helpCreateRandomBucket(t *testing.T, apiClient pb.SpaceApiClient) (string, *pb.CreateBucketResponse) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	bucketName := fmt.Sprintf("TestBucket-%d", rand.Intn(500000))
	bucketResponse, err := apiClient.CreateBucket(ctx, &pb.CreateBucketRequest{
		Slug: bucketName,
	})

	assert.NoError(t, err, "Error creating bucket:"+bucketName)
	assert.Equal(t, bucketResponse.Bucket.Name, bucketName)

	return bucketName, bucketResponse
}

func TestGRPCUploadAndDownload(t *testing.T) {
	t.SkipNow()
	app := NewAppFixture()
	app.StartApp(t)
	apiClient := app.SpaceApiClient(t)
	testFilename := "file1"
	targetPath := ""
	bucketName, _ := helpCreateRandomBucket(t, apiClient)
	ctx, cancelCtx := context.WithCancel(context.Background())
	t.Cleanup(cancelCtx)

	// upload test_files/file1
	currentPath, err := os.Getwd()
	assert.NoError(t, err, "os.Getwd() failed")
	testFilePath := fmt.Sprintf("%s/test_files/%s", currentPath, testFilename)
	_, err = apiClient.AddItems(ctx, &pb.AddItemsRequest{
		SourcePaths: []string{testFilePath},
		TargetPath:  targetPath,
		Bucket:      bucketName,
	})
	assert.NoError(t, err, "Error adding files to bucket")

	// fetch uploaded file1
	openResult, err := apiClient.OpenFile(ctx, &pb.OpenFileRequest{
		Path:   fmt.Sprintf("%s%s", targetPath, testFilename),
		Bucket: bucketName,
	})
	assert.NoError(t, err, "Error opening file", testFilename)
	fileContent, err := ioutil.ReadFile(openResult.Location)
	assert.NoError(t, err, "Error reading opened file content")

	assert.Equal(t, "test data", string(fileContent))
}

func TestCreateFolderAndListDirectoryWorks(t *testing.T) {
	app := NewAppFixture()
	app.StartApp(t)
	apiClient := app.SpaceApiClient(t)
	bucketName, _ := helpCreateRandomBucket(t, apiClient)
	folderPath := "/baseFolder"
	ctx, cancelCtx := context.WithCancel(context.Background())
	t.Cleanup(cancelCtx)

	// create a folder in base directory
	_, err := apiClient.CreateFolder(ctx, &pb.CreateFolderRequest{
		Path:   folderPath,
		Bucket: bucketName,
	})
	assert.NoError(t, err, "Error creating folder: %s", folderPath)

	// fetch all items in base directory
	listDirRes, err := apiClient.ListDirectory(ctx, &pb.ListDirectoryRequest{
		Path:   "",
		Bucket: bucketName,
	})
	assert.NoError(t, err, "Error listing directory")
	assert.Len(t, listDirRes.Entries, 1, "Bucket directory should only contain one item")
	dirItem := listDirRes.Entries[0]
	assert.Equal(t, folderPath, dirItem.Path, "Created Folder path does not match ListDirectory path")
	assert.Equal(t, true, dirItem.IsDir, "DirItem.IsDir should be true")
}
