package services

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/space/domain"
)

type createIdentityRequest struct {
	PublicKey string `json:"publicKey"`
	Username  string `json:"username"`
}

func parseIdentity(resp *http.Response) (*domain.Identity, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		var returnedErr domain.APIError
		err = json.Unmarshal(body, &returnedErr)
		if err != nil {
			return nil, err
		}

		if returnedErr.Message != "" {
			return nil, errors.New(returnedErr.Message)
		}

		return nil, errors.New("Unexpected API error")
	}

	var newIdentity domain.Identity
	err = json.Unmarshal(body, &newIdentity)
	if err != nil {
		return nil, err
	}

	return &newIdentity, nil

}

// Creates an identity in Space cloud services. Returns the created identity or an error if any.
func (s *Space) CreateIdentity(ctx context.Context, username string) (*domain.Identity, error) {
	_, pub, err := s.keychain.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return nil, err
	}

	publicKeyBytes, err := pub.Raw()
	if err != nil {
		return nil, err
	}

	publicKeyHex := hex.EncodeToString(publicKeyBytes)

	identity := &createIdentityRequest{
		PublicKey: publicKeyHex,
		Username:  username,
	}
	identityJSON, err := json.Marshal(identity)
	if err != nil {
		return nil, err
	}
	apiURL := s.cfg.GetString(config.SpaceServicesAPIURL, "")
	resp, err := http.Post(
		apiURL+"/identities",
		"application/json",
		bytes.NewBuffer(identityJSON),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseIdentity(resp)
}

// Gets an identity from Space cloud services given a username
func (s *Space) GetIdentityByUsername(ctx context.Context, username string) (*domain.Identity, error) {
	apiURL := s.cfg.GetString(config.SpaceServicesAPIURL, "")
	resp, err := http.Get(apiURL + "/identities/username/" + username)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseIdentity(resp)
}
