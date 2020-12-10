package integration_tests

import (
	"context"
	"path/filepath"

	"github.com/FleekHQ/space-daemon/integration_tests/fixtures"

	"github.com/FleekHQ/space-daemon/grpc/pb"
	. "github.com/FleekHQ/space-daemon/integration_tests/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("App Uploads", func() {
	It("should create empty folder successfully", func() {
		ctx := context.Background()
		folderName := fixtures.RandomPathName()

		CreateEmptyFolder(ctx, app.Client(), folderName)
		_, err := app.Client().ListDirectory(ctx, &pb.ListDirectoryRequest{
			Path:   "",
			Bucket: fixtures.DefaultBucket,
		})
		Expect(err).NotTo(HaveOccurred())
		ExpectFileExists(ctx, app.Client(), "", folderName)
	})

	It("should upload and download files successfully", func() {
		ctx := context.Background()
		file := CreateLocalStringFile("random file content")
		fileName := filepath.Base(file.Name())

		UploadFilesToTargetPath(ctx, app.Client(), "", []string{file.Name()})
		ExpectFileExists(ctx, app.Client(), "", fileName)

		// try uploading to a folder
		topFolderPath := fixtures.RandomPathName()
		CreateEmptyFolder(ctx, app.Client(), topFolderPath)
		UploadFilesToTargetPath(ctx, app.Client(), topFolderPath, []string{file.Name()})
		ExpectFileExists(ctx, app.Client(), topFolderPath, fileName)
	})
})
