package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReceivedFileSchema_IsPublicLinkReceived_ShouldBeFalse_For_InvitationId(t *testing.T) {
	schema := ReceivedFileSchema{
		ReceivedFileViaInvitationSchema: ReceivedFileViaInvitationSchema{
			DbID:          "some-db-id",
			Bucket:        "personal-mirror",
			Path:          "/",
			InvitationId:  "some-invitation-id",
			BucketKey:     "",
			EncryptionKey: []byte(""),
		},
	}

	assert.False(t, schema.IsPublicLinkReceived(), "received file should not be public")
}
