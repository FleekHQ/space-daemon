package ipfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ipfs/go-cid"
	chunker "github.com/ipfs/go-ipfs-chunker"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs/importer/balanced"
	"github.com/ipfs/go-unixfs/importer/helpers"
	"github.com/ipfs/go-unixfs/importer/trickle"
	mh "github.com/multiformats/go-multihash"
)

func GetFileHash(r io.Reader) (string, error) {
	hashFun := "sha2-256"

	prefix, err := merkledag.PrefixForCidVersion(1)
	if err != nil {
		return "", fmt.Errorf("bad CID Version: %s", err)
	}

	hashFunCode, ok := mh.Names[strings.ToLower(hashFun)]
	if !ok {
		return "", fmt.Errorf("unrecognized hash function: %s", hashFun)
	}
	prefix.MhType = hashFunCode
	prefix.MhLength = -1
	prefix.Codec = cid.DagProtobuf

	dagServ := NewDagService()
	dbp := helpers.DagBuilderParams{
		Dagserv:    dagServ,
		RawLeaves:  true,
		Maxlinks:   helpers.DefaultLinksPerBlock,
		NoCopy:     false,
		CidBuilder: &prefix,
	}

	chnk, err := chunker.FromString(r, "")
	if err != nil {
		return "", err
	}
	dbh, err := dbp.New(chnk)
	if err != nil {
		return "", err
	}

	layout := "trickle"
	var n ipld.Node
	switch layout {
	case "trickle":
		n, err = trickle.Layout(dbh)
	case "balanced", "":
		n, err = balanced.Layout(dbh)
	default:
		return "", errors.New("invalid Layout")
	}

	return n.Cid().String(), nil

}

func DownloadIpfsItemViaGateway(ctx context.Context, gatewayUrl string, cid cid.Cid) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/ipfs/%s", gatewayUrl, cid.String())
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch item %s: status_code %d", cid.String(), resp.StatusCode)
	}

	return resp.Body, nil
}

func DownloadIpfsItem(ctx context.Context, nodeUrl string, cid cid.Cid) (io.ReadCloser, error) {

	// https://docs.ipfs.io/reference/http/api/#api-v0-cat
	url := fmt.Sprintf("%s/api/v0/cat?arg=%s", nodeUrl, cid.String())

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch item %s: status_code %d", cid.String(), resp.StatusCode)
	}

	return resp.Body, nil
}
