package helpers

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/onsi/ginkgo"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/FleekHQ/space-daemon/integration_tests/fixtures"

	"github.com/FleekHQ/space-daemon/grpc/pb"
	. "github.com/onsi/gomega"
)

func CreateEmptyFolder(ctx context.Context, client pb.SpaceApiClient, path string) {
	_, err := client.CreateFolder(ctx, &pb.CreateFolderRequest{
		Path:   path,
		Bucket: fixtures.DefaultBucket,
	})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	<-time.After(4 * time.Second) // required currently for textile to perform sync properly. TODO: Fix this
}

func CreateLocalStringFile(strContent string) *os.File {
	content := []byte(strContent)
	tmpfile, err := ioutil.TempFile("", "*-localStringFile.txt")
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to create local string file")
	if err != nil {
		defer tmpfile.Close()
	}

	_, err = tmpfile.Write(content)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to write string content")

	return tmpfile
}

func UploadFilesToTargetPath(
	ctx context.Context,
	client pb.SpaceApiClient,
	targetPath string,
	sourcePaths []string,
) {
	streamResponse, err := client.AddItems(ctx, &pb.AddItemsRequest{
		SourcePaths: sourcePaths,
		TargetPath:  targetPath,
		Bucket:      fixtures.DefaultBucket,
	})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	_, err = streamResponse.Recv()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	<-time.After(4 * time.Second) // required currently for textile to perform sync properly. TODO: Fix this
}

// StartWatchingForRemoteUploads kicks off a goroutine that subscribes to Subscribe rpc for the client
// and watches for the specified files and buckets to be uploaded
// The first channels returns returns a true when all files have been found or false after a 5 minutes timeout of not finding them all
// the second function returned must always be called at the end of a test to ensure the goroutine is stopped.
func StartWatchingForRemoteBackup(
	ctx context.Context,
	client pb.SpaceApiClient,
	filePathsToWatch []string,
) (<-chan bool, func()) {
	ExpectWithOffset(1, filePathsToWatch).NotTo(BeEmpty())

	streamCtx, cancelStreamCtx := context.WithCancel(context.Background())
	streamResponse, err := client.Subscribe(streamCtx, &empty.Empty{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Subscribe failed to connect")
	foundFiles := make(map[string]bool)
	totalFound := 0
	foundAllChan := make(chan bool, 1)
	closeAChan := make(chan bool, 1)
	itemsChan := make(chan *pb.FileEventResponse)

	go func() {
		defer ginkgo.GinkgoRecover()

		timeoutChan := time.After(5 * time.Minute)
		for {
			select {
			case <-closeAChan:
				return
			case <-timeoutChan:
				foundAllChan <- totalFound == len(filePathsToWatch)
				return
			case item := <-itemsChan:
				if !foundFiles[item.Entry.Path] && item.Bucket == fixtures.DefaultBucket && item.Type == pb.EventType_ENTRY_BACKUP_READY {
					for _, path := range filePathsToWatch {
						if path == item.Entry.Path {
							foundFiles[item.Entry.Path] = true
							totalFound++
						}
					}
				}

				if totalFound == len(filePathsToWatch) {
					// wait for completion
					foundAllChan <- true // all file have been found
					return
				}
			}
		}
	}()

	go func() {
		defer ginkgo.GinkgoRecover()

		for {
			select {
			default:
				item, err := streamResponse.Recv()
				if err == io.EOF { // stream closed
					return
				}
				ExpectWithOffset(2, err).NotTo(HaveOccurred(), "failed while receiving data from stream")
				if err == nil {
					itemsChan <- item
				}
			}
		}
	}()

	return foundAllChan, func() {
		closeAChan <- true
		cancelStreamCtx()
	}
}

// i need to be receiving and in parallel
