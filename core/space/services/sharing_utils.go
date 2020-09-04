package services

import (
	"encoding/hex"
	"errors"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	crypto "github.com/libp2p/go-libp2p-crypto"
)

var EmptyFileSharingInfo = domain.FileSharingInfo{}

func generateFilesSharingZip() string {
	//return fmt.Sprintf("space_shared_files-%d.zip", time.Now().UnixNano())
	return "space_shared_files.zip"
}

func extractInvitation(notification *domain.Notification) (domain.Invitation, error) {
	if notification.NotificationType != domain.INVITATION {
		return domain.Invitation{}, errInvitationNotFound
	}

	return notification.InvitationValue, nil
}

// NOTE: This assumes that the public key string is ed25519 hex encoded string
func decodePublicKey(err error, pkString string) (crypto.PubKey, error) {
	pkBytes, err := hex.DecodeString(pkString)
	if err != nil {
		return nil, errors.New("invalid encoding for public key")
	}

	pk, err := crypto.UnmarshalEd25519PublicKey(pkBytes)
	if err != nil {
		return nil, errors.New("invalid public key format")
	}
	return pk, nil
}
