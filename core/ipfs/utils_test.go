package ipfs

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIpfs_GetFileHash(t *testing.T) {
	r := strings.NewReader("IPFS test data for reader")
	expectedHash := "bafybeie4zu4wu7lexqty2aubpe36dnpd6edgb5mthhtab5hyhuju7jlcgm"
	res, err := GetFileHash(r)
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, *res, expectedHash)
}
