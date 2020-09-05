package textile

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/log"
	crypto "github.com/libp2p/go-libp2p-crypto"
)

func (tc *textileClient) ShareFilesViaPublicKey(ctx context.Context, paths []domain.FullPath, pubkeys []crypto.PubKey) error {
	var err error
	ctx, err = tc.getHubCtx(ctx)
	if err != nil {
		return err
	}

	for _, pth := range paths {
		// TODO: uncomment once mirror bucket setup is done
		// ctx, _, err = tc.getBucketContext(ctx, mirror.DbId, mirror.Bucket, true)
		// if err != nil {
		// 	return err
		// }

		log.Info("Adding roles for pth: " + pth.Path)
		// TOOD: uncomment once release and upgraded txl pkg
		// var roles map[string]buckets.Role
		// for _, pk := range pubkeys {
		// 	roles[pk] = buckets.Role.Writer
		// }
		// err := tc.PushPathAccessRoles(ctx, key, pth, roles)
		// if err != nil {
		//   return err
		// }
	}

	return nil
}

var errInvitationNotPending = errors.New("invitation is no more pending")
var errInvitationAlreadyAccepted = errors.New("invitation is already accepted")
var errInvitationAlreadyRejected = errors.New("invitation is already rejected")

func (tc *textileClient) AcceptSharedFilesInvitation(
	ctx context.Context,
	invitation domain.Invitation,
) (domain.Invitation, error) {
	if invitation.Status == domain.ACCEPTED {
		return domain.Invitation{}, errInvitationAlreadyAccepted
	}

	if invitation.Status != domain.PENDING {
		return domain.Invitation{}, errInvitationNotPending
	}

	err := tc.createReceivedFiles(ctx, invitation.InvitationID, true, invitation.ItemPaths)
	if err != nil {
		return domain.Invitation{}, err
	}
	invitation.Status = domain.ACCEPTED

	return invitation, nil
}

func (tc *textileClient) RejectSharedFilesInvitation(
	ctx context.Context,
	invitation domain.Invitation,
) (domain.Invitation, error) {
	if invitation.Status == domain.REJECTED {
		return domain.Invitation{}, errInvitationAlreadyRejected
	}

	if invitation.Status != domain.PENDING {
		return domain.Invitation{}, errInvitationNotPending
	}

	err := tc.createReceivedFiles(ctx, invitation.InvitationID, false, invitation.ItemPaths)
	if err != nil {
		return domain.Invitation{}, err
	}
	invitation.Status = domain.REJECTED

	return invitation, nil
}

func (tc *textileClient) createReceivedFiles(
	ctx context.Context,
	invitationId string,
	accepted bool,
	paths []domain.FullPath,
) error {
	// TODO: Make this is call a transaction on threads so any failure can be easily reverted

	var allErr error
	for _, path := range paths {
		_, err := tc.GetModel().CreateReceivedFile(ctx, path, invitationId, accepted)

		// compose each create error
		if err != nil {
			if allErr == nil {
				allErr = errors.New("Failed to accept some invitations")
			}
			allErr = errors.Wrap(allErr, "")
		}
	}

	return allErr
}

func (tc *textileClient) GetReceivedFiles(ctx context.Context, accepted bool, seek string, limit int) ([]*domain.SharedDirEntry, string, error) {
	files, err := tc.GetModel().ListReceivedFiles(ctx, accepted, seek, limit)
	if err != nil {
		return nil, "", err
	}

	items := []*domain.SharedDirEntry{}

	for _, file := range files {
		// TODO: Get these fields from mirror bucket
		ipfsHash := ""
		name := ""
		isDir := false
		size := ""
		ext := ""

		res := &domain.SharedDirEntry{
			Bucket: file.Bucket,
			DbID:   file.DbID,
			FileInfo: domain.FileInfo{
				IpfsHash: ipfsHash,
				DirEntry: domain.DirEntry{
					Path:          file.Path,
					IsDir:         isDir,
					Name:          name,
					SizeInBytes:   size,
					FileExtension: ext,
					Created:       time.Unix(file.CreatedAt, 0).String(),
					Updated:       time.Unix(file.CreatedAt, 0).String(),
				},
			},
		}

		items = append(items, res)
	}

	offset := files[len(files)-1].ID.String()

	return items, offset, nil
}
