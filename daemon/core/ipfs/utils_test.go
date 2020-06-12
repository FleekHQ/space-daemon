package ipfs

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fleek hash: bafybeiemzcxynbrrhtcpmmdtkl42molkiyfqu3j5ewp2o7izdmomptfkgi

func TestIpfs_GetFileHash_FromStringReader(t *testing.T) {
	t.Skip()
	r := strings.NewReader("IPFS test data for reader")
	expectedHash := "bafybeie4zu4wu7lexqty2aubpe36dnpd6edgb5mthhtab5hyhuju7jlcgm"

	res, err := GetFileHash(r)

	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, expectedHash, res)
}

// bafybeic3jetthfk7tjmewz42idwsaeek5a7myw6n46zrrxdmp5nlkc6diy

func TestIpfs_GetFileHash_FromFile(t *testing.T) {
	t.Skip()
	r, _ := os.Open("test1.txt")
	expectedHash := "bafybeie4zu4wu7lexqty2aubpe36dnpd6edgb5mthhtab5hyhuju7jlcgm"

	res, err := GetFileHash(r)

	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, expectedHash, res)
}


