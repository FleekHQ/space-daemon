package initializer

import (
	tc "github.com/FleekHQ/space-poc/core/textile/client"
	"github.com/FleekHQ/space-poc/log"
)

const defaultPersonalBucketSlug = "personal"

// Checks that the basic resources such as the default bucket exists.
// Creates it if it doesn't exist.
// Requires a textile client that has already been started.
func InitialLoad(textileClient *tc.TextileClient) error {
	if err := textileClient.Start(); err != nil {
		log.Error("error starting textile client", err)
		return err
	}

	if err := textileClient.CreateBucket(defaultPersonalBucketSlug); err != nil {
		log.Error("error creating default bucket", err)
		return err
	}

	// TODO: Add other initialization logic
	return nil
}
