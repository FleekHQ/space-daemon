package textile

import (
	"context"

	crypto "github.com/libp2p/go-libp2p-crypto"
)

func (tc *textileClient) ShareFilesViaPublicKey(ctx context.Context, bucketName string, paths []string, pubkeys []crypto.PubKey) error {
	// TOOD: uncomment once release and upgraded txl pkg
	// ctx := tc.getHubCtx(ctx)

	// for _, pth := range paths {
	// 	var roles map[string]buckets.Role
	// 	for _, pk := range pubkeys {
	// 		roles[pk] = buckets.Role.Writer
	// 	}
	// 	err := tc.EditPathAccessRoles(ctx, key, pth, roles)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	return nil
}
