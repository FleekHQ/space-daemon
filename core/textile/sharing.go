package textile

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	crypto "github.com/libp2p/go-libp2p-crypto"
)

func (tc *textileClient) ShareFilesViaPublicKey(ctx context.Context, paths []domain.FullPath, pubkeys []crypto.PubKey) error {
	// TOOD: uncomment once release and upgraded txl pkg
	// ctx = tc.getHubCtx(ctx)

	// for _, pth := range paths {
	//   ctx = tc.GetRemoteBucketContext(ctx, pth.DbId, pth.Bucket)
	//   var roles map[string]buckets.Role
	//   for _, pk := range pubkeys {
	//     roles[pk] = buckets.Role.Writer
	//   }
	//   err := tc.PushPathAccessRoles(ctx, key, pth, roles)
	//   if err != nil {
	//     return err
	//   }
	// }

	return nil
}
