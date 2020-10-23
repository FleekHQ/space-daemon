package helpers

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/FleekHQ/space-daemon/integration_tests/fixtures"

	"github.com/FleekHQ/space-daemon/grpc/pb"
	. "github.com/onsi/gomega"
)

func ExpectFileExists(ctx context.Context, client pb.SpaceApiClient, remotePath string, remoteFileName string) *pb.ListDirectoryEntry {
	res, err := client.ListDirectory(ctx, &pb.ListDirectoryRequest{
		Path:   remotePath,
		Bucket: fixtures.DefaultBucket,
	})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, res.Entries).NotTo(BeEmpty(), "file at remote path not found")

	for _, item := range res.Entries {
		if item.Name == remoteFileName {
			return item
		}
	}
	// Not Found
	ExpectWithOffset(1, errors.New("file at remote path not found")).NotTo(HaveOccurred())
	return nil
}

func ExpectFileContentEquals(filePath string, expectedContent []byte) {
	actualContent, err := ioutil.ReadFile(filePath)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, actualContent).To(Equal(expectedContent))
}

func ExpectFileToBeSharedWithMe(
	ctx context.Context,
	client pb.SpaceApiClient,
	fileName, bucket, dbId string,
	isPublicShared bool,
) *pb.SharedListDirectoryEntry {
	// assert file is visible in recently shared list
	sharedWithMeRes, err := client.GetSharedWithMeFiles(ctx, &pb.GetSharedWithMeFilesRequest{Limit: 10})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, sharedWithMeRes.Items).NotTo(BeEmpty())
	for _, item := range sharedWithMeRes.Items {
		if item.Entry.Name == fileName {
			ExpectWithOffset(1, item.IsPublicLink).To(Equal(isPublicShared), "shared files is publicly shared not expected value")
			ExpectWithOffset(1, item.Bucket).To(Equal(bucket), "shared files bucket slug not expected value")
			if dbId != "<ignore>" { // conditionally skip this for some checks
				ExpectWithOffset(1, item.DbId).To(Equal(dbId), "shared files dbId not expected value")
			}
			return item
		}
	}
	// Not Found
	ExpectWithOffset(1, errors.New(fmt.Sprintf("shared with me file not found. filename=%s, isPublicShared=%v", fileName, isPublicShared)))
	return nil
}
