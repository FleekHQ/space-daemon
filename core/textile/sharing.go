package textile

import (
	"context"

	crypto "github.com/libp2p/go-libp2p-crypto"
)

func (tc *textileClient) ShareFilesViaPublicKey(ctx context.Context, bucketName string, paths []string, pubkeys []crypto.PubKey) error {
	return nil
}
