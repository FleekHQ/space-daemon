package textile

import "context"

func (tc *textileClient) initSearchIndex(ctx context.Context) error {
	err := tc.GetModel().InitSearchIndexCollection(ctx)
	return err
}
