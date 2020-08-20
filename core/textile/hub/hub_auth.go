package hub

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/log"
	threadsClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/textile/api/common"
	"golang.org/x/net/websocket"
)

type sentMessageData struct {
	Signature []byte `json:"sig"`
	PublicKey string `json:"pubkey"`
}

type outMessage struct {
	Action string          `json:"action"`
	Data   sentMessageData `json:"data"`
}

type inMessageChallengeValue struct {
	Type string `json:"type"`
	Data []byte `json:"data"`
}

type inMessageChallenge struct {
	Type  string                  `json:"type"`
	Value inMessageChallengeValue `json:"value"`
}

type inMessageTokenValue struct {
	Token string `json:"token"`
}

type inMessageToken struct {
	Type  string              `json:"type"`
	Value inMessageTokenValue `json:"value"`
}

const hubTokenStoreKey = "hubAuthToken"

func getHubTokenFromStore(st store.Store) (string, error) {
	key := []byte(hubTokenStoreKey)
	val, _ := st.Get(key)

	if val == nil {
		return "", nil
	}

	return string(val), nil
}

func storeHubToken(st store.Store, hubToken string) error {
	err := st.Set([]byte(hubTokenStoreKey), []byte(hubToken))

	return err
}

func GetHubToken(ctx context.Context, st store.Store, kc keychain.Keychain, cfg config.Config) (string, error) {
	// Try to avoid redoing challenge if we already have the token
	if valFromStore, err := getHubTokenFromStore(st); err != nil {
		return "", err
	} else if valFromStore != "" {
		log.Debug("Got hub token from store: " + valFromStore)
		return valFromStore, nil
	}

	log.Debug("Token Challenge: Connecting through websocket")
	conn, err := websocket.Dial(cfg.GetString(config.SpaceServicesHubAuthURL, ""), "", "http://localhost/")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	log.Debug("Token Challenge: Connected")

	privateKey, _, err := kc.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return "", err
	}

	identity := thread.NewLibp2pIdentity(privateKey)
	pub := identity.GetPublic().String()

	// Request a challenge (a payload we need to sign)
	log.Debug("Token Challenge: Sending token request with pub key" + pub)
	tokenRequest := &outMessage{
		Action: "token",
		Data: sentMessageData{
			PublicKey: identity.GetPublic().String(),
		},
	}
	err = websocket.JSON.Send(conn, tokenRequest)
	if err != nil {
		return "", err
	}

	challenge := inMessageChallenge{}
	if err := websocket.JSON.Receive(conn, &challenge); err != nil {
		return "", err
	}
	log.Debug("Token Challenge: Received challenge")

	solution, err := identity.Sign(ctx, challenge.Value.Data)
	if err != nil {
		return "", err
	}

	// Send back channel solution
	solMessage := &outMessage{
		Action: "challenge",
		Data: sentMessageData{
			Signature: solution,
			PublicKey: pub,
		},
	}
	log.Debug("Token Challenge: Sending signature")
	err = websocket.JSON.Send(conn, solMessage)
	if err != nil {
		return "", err
	}

	// Receive the token
	var token inMessageToken
	for token.Type != "token" {
		currToken := inMessageToken{}
		if err := websocket.JSON.Receive(conn, &token); err != nil {
			return "", err
		}

		if currToken.Type == "token" {
			token = currToken
		}
	}
	log.Debug("Token Challenge: Received token successfully")

	if err := storeHubToken(st, token.Value.Token); err != nil {
		return "", err
	}

	return token.Value.Token, nil
}

// This method is just for testing purposes. Keys shouldn't be bundled in the daemon.
// Use GetHubToken instead.
func GetHubTokenUsingTextileKeys(ctx context.Context, st store.Store, kc keychain.Keychain, threads *threadsClient.Client) (context.Context, error) {
	var tokStr string

	// prebuild context, needs to happen
	// whether token is saved or not
	key := os.Getenv("TXL_USER_KEY")
	secret := os.Getenv("TXL_USER_SECRET")

	if key == "" || secret == "" {
		return nil, errors.New("Couldn't get Textile key or secret from envs")
	}
	ctx = common.NewAPIKeyContext(ctx, key)

	apiSigCtx, err := common.CreateAPISigContext(ctx, time.Now().Add(time.Minute), secret)
	if err != nil {
		return nil, err
	}
	ctx = apiSigCtx

	privateKey, _, err := kc.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return nil, err
	}

	// Try to avoid redoing challenge if we already have the token
	if tokStr, err = getHubTokenFromStore(st); err != nil {
		return nil, err
	} else if tokStr == "" {

		tok, err := threads.GetToken(ctx, thread.NewLibp2pIdentity(privateKey))
		if err != nil {
			return nil, err
		}

		tokStr = string(tok)

		if err := storeHubToken(st, tokStr); err != nil {
			return nil, err
		}
	}

	newtok := thread.Token(tokStr)
	ctx = thread.NewTokenContext(ctx, newtok)
	return ctx, nil
}
