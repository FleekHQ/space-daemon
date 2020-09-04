package textile

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/log"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/textileio/textile/buckets"
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
		var roles map[string]buckets.Role
		for _, pk := range pubkeys {
			pkb, err := pk.Bytes()
			if err != nil {
				return err
			}
			roles[string(pkb)] = buckets.Writer
		}
		// TODO: replace key with actual key from remote bucket
		err := tc.hb.PushPathAccessRoles(ctx, "key", pth.Path, roles)
		if err != nil {
			return err
		}
	}

	return nil
}
