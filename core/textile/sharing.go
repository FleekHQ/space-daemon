package textile

import (
	"context"

	"github.com/pkg/errors"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/log"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/textileio/textile/buckets"
)

func (tc *textileClient) ShareFilesViaPublicKey(ctx context.Context, paths []domain.FullPath, pubkeys []crypto.PubKey, keys [][]byte) error {
	var err error
	ctx, err = tc.getHubCtx(ctx)
	if err != nil {
		return err
	}

	for i, pth := range paths {
		ctx, _, err = tc.getBucketContext(ctx, pth.DbId, pth.Bucket, true, keys[i])
		if err != nil {
			return err
		}

		log.Info("Adding roles for pth: " + pth.Path)
		roles := make(map[string]buckets.Role)
		for _, pk := range pubkeys {
			pkb, err := pk.Bytes()
			if err != nil {
				return err
			}
			roles[string(pkb)] = buckets.Writer
		}

		err := tc.hb.PushPathAccessRoles(ctx, pth.BucketKey, pth.Path, roles)
		if err != nil {
			return err
		}
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

	err := tc.createReceivedFiles(ctx, invitation.InvitationID, true, invitation.ItemPaths, invitation.Keys)
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

	err := tc.createReceivedFiles(ctx, invitation.InvitationID, false, invitation.ItemPaths, [][]byte{})
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
	keys [][]byte,
) error {
	// TODO: Make this is call a transaction on threads so any failure can be easily reverted

	var allErr error
	for i, path := range paths {
		_, err := tc.GetModel().CreateReceivedFile(ctx, path, invitationId, accepted, keys[i])

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
