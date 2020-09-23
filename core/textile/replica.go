package textile

import (
	"context"
	"fmt"

	"github.com/FleekHQ/space-daemon/config"

	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/textile/cmd"
)

// replicate a local thread on the hub
func (tc *textileClient) ReplicateThreadToHub(ctx context.Context, dbID *thread.ID) error {

	hubma := tc.cfg.GetString(config.TextileHubMa, "")
	if hubma == "" {
		return fmt.Errorf("no textile hub set")
	}

	_, err := tc.netc.AddReplicator(ctx, *dbID, cmd.AddrFromStr(hubma))
	if err != nil {
		return err
	}

	return nil
}

// dereplicate a local thread from the hub
func (tc *textileClient) DereplicateThreadFromHub(ctx context.Context, dbID *thread.ID) error {

	// TODO

	return nil
}
