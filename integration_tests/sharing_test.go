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

var _ = Describe("Sharing Files", func() {
	Context("when sharing publicly", func() {
		It("should work", func() {
			ctx := context.Background()
			password := "random-strong-password"
			sharedFileContent := "Perhaps really long text"
			file := CreateLocalStringFile(sharedFileContent)
			fileName := filepath.Base(file.Name())

			// share file
			UploadFilesToTargetPath(ctx, app.Client(), "", []string{file.Name()})
			publicLinkRes, err := app.Client().GeneratePublicFileLink(ctx, &pb.GeneratePublicFileLinkRequest{
				Bucket:    fixtures.DefaultBucket,
				ItemPaths: []string{fileName},
				Password:  password,
			})
			Expect(err).NotTo(HaveOccurred())

			// fetch shared file
			openLinkRes, err := app.Client().OpenPublicFile(ctx, &pb.OpenPublicFileRequest{
				FileCid:  publicLinkRes.FileCid,
				Password: password,
				Filename: fileName,
			})
			Expect(err).NotTo(HaveOccurred())
			ExpectFileContentEquals(openLinkRes.Location, []byte(sharedFileContent))

			// assert file is visible in recently shared list
			ExpectFileToBeSharedWithMe(ctx, app.Client(), fileName, "", "", true)
		})
	})

	Context("when sharing privately", func() {
		var app2 *fixtures.RunAppCtx

		BeforeEach(func() {
			app2 = fixtures.RunAppWithClientAppToken(app.ClientAppToken)
			InitializeApp(app2)
		})

		AfterEach(func() {
			app2.Shutdown()
		})

		It("should work", func() {
			ctx := context.Background()
			sharedFileContent := "A really really really long text or possibly binary data"
			file := CreateLocalStringFile(sharedFileContent)
			fileName := filepath.Base(file.Name())
			pk1Res, err := app.Client().GetPublicKey(ctx, &pb.GetPublicKeyRequest{})
			Expect(err).NotTo(HaveOccurred())

			// upload file to second users directory
			remoteBackupWait, cleanupWait := StartWatchingForRemoteBackup(ctx, app2.Client(), []string{fileName})
			defer cleanupWait()
			UploadFilesToTargetPath(ctx, app2.Client(), "", []string{file.Name()})

			// Wait for file to be replicated to the hub
			backupSuccess := <-remoteBackupWait
			Expect(backupSuccess).To(BeTrue(), "failed to complete remote backup of watched files")

			// share file with the first user
			_, err = app2.Client().ShareFilesViaPublicKey(ctx, &pb.ShareFilesViaPublicKeyRequest{
				PublicKeys: []string{pk1Res.PublicKey},
				Paths: []*pb.FullPath{{
					Bucket: fixtures.DefaultBucket,
					Path:   fileName,
				}},
			})
			Expect(err).NotTo(HaveOccurred())

			// Fetch and verify invite notification.
			notifRes, err := app.Client().GetNotifications(ctx, &pb.GetNotificationsRequest{Limit: 10})
			Expect(err).NotTo(HaveOccurred())
			Expect(notifRes.Notifications).NotTo(BeEmpty(), "no invite notification provided")
			Expect(notifRes.Notifications[0].Type).To(Equal(pb.NotificationType_INVITATION))

			// Accept invite
			_, err = app.Client().HandleFilesInvitation(ctx, &pb.HandleFilesInvitationRequest{
				InvitationID: notifRes.Notifications[0].ID,
				Accept:       true,
			})
			Expect(err).NotTo(HaveOccurred())

			// verify file is in shared with first user
			sharedItem := ExpectFileToBeSharedWithMe(ctx, app.Client(), fileName, fixtures.MirrorBucket, "<ignore>", false)

			// confirm first user can see file
			openFileResult, err := app.Client().OpenFile(ctx, &pb.OpenFileRequest{
				Path:   sharedItem.Entry.Path,
				Bucket: sharedItem.Bucket,
				DbId:   sharedItem.DbId,
			})
			Expect(err).NotTo(HaveOccurred())
			ExpectFileContentEquals(openFileResult.Location, []byte(sharedFileContent))
		})
	})
})
