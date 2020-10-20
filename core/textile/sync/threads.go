package sync

import (
	"context"
	"fmt"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/textile/v2/cmd"
)

// replicate a local thread on the hub
func (s *synchronizer) replicateThreadToHub(ctx context.Context, dbID *thread.ID) error {

	hubma := s.cfg.GetString(config.TextileHubMa, "")
	if hubma == "" {
		return fmt.Errorf("no textile hub set")
	}

	_, err := s.netc.AddReplicator(ctx, *dbID, cmd.AddrFromStr(hubma))
	if err != nil {
		return err
	}

	return nil
}

// dereplicate a local thread from the hub
func (s *synchronizer) dereplicateThreadFromHub(ctx context.Context, dbID *thread.ID) error {

	// TODO

	return nil
}
