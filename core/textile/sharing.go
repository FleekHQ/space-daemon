package textile

import (
	"context"

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
		ctx, _, err = tc.getBucketContext(ctx, pth.DbId, pth.Bucket, true)
		if err != nil {
			return err
		}

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
