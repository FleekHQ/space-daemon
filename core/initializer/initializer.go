package initializer

import (
	tc "github.com/FleekHQ/space-poc/core/textile/client"
	"github.com/FleekHQ/space-poc/log"
)

const defaultPersonalBucketSlug = "personal"

// Checks that the basic resources such as the default bucket exists.
// Creates it if it doesn't exist.
// Requires a textile client that has already been started.
func InitialLoad(textileClient *tc.TextileClient) {
	if err := textileClient.Start(); err != nil {
		log.Error("error starting textile client", err)
		return
	}

	if err := textileClient.CreateBucket(defaultPersonalBucketSlug); err != nil {
		log.Error("Error when creating default bucket. Maybe it already exists", err)
	}

	// TODO: Add other initialization logic
}
