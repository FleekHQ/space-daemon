package hub

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/log"
	mbase "github.com/multiformats/go-multibase"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/textile/api/common"
	"golang.org/x/net/websocket"
)

type sentMessageData struct {
	Signature string `json:"sig"`
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
	Token    string `json:"token"`
	Key      string `json:"key"`
	Msg      string `json:"msg"`
	Sig      string `json:"sig"`
	AppToken string `json:"appToken"`
}

type inMessageToken struct {
	Type  string              `json:"type"`
	Value inMessageTokenValue `json:"value"`
}

type AuthTokens struct {
	HubToken string
	Key      string
	Sig      string
	AppToken string
	Msg      string
}

const tokensStoreKey = "hubTokens"

func retrieveTokens(st store.Store) (*inMessageTokenValue, error) {
	stored, err := st.Get([]byte(tokensStoreKey))
	if err != nil {
		return nil, err
	}

	tokens := &inMessageTokenValue{}
	tokensBytes := bytes.NewBuffer(stored)

	if err := json.NewDecoder(tokensBytes).Decode(tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

func storeTokens(st store.Store, tokens *inMessageTokenValue) error {
	tokensBytes := new(bytes.Buffer)
	if err := json.NewEncoder(tokensBytes).Encode(tokens); err != nil {
		return err
	}

	if err := st.Set([]byte(tokensStoreKey), tokensBytes.Bytes()); err != nil {
		return err
	}

	return nil
}

func getTokensThroughChallenge(ctx context.Context, kc keychain.Keychain, cfg config.Config) (*inMessageTokenValue, error) {
	log.Debug("Token Challenge: Connecting through websocket")
	conn, err := websocket.Dial(cfg.GetString(config.SpaceServicesHubAuthURL, ""), "", "http://localhost/")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	log.Debug("Token Challenge: Connected")

	privateKey, _, err := kc.GetStoredKeyPairInLibP2PFormat()

	if err != nil {
		return nil, err
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
		return nil, err
	}

	challenge := inMessageChallenge{}
	if err := websocket.JSON.Receive(conn, &challenge); err != nil {
		return nil, err
	}
	log.Debug("Token Challenge: Received challenge")

	solution, err := identity.Sign(ctx, challenge.Value.Data)

	if err != nil {
		return nil, err
	}

	signature := b64.StdEncoding.EncodeToString(solution)

	// Send back channel solution
	solMessage := &outMessage{
		Action: "challenge",
		Data: sentMessageData{
			Signature: signature,
			PublicKey: pub,
		},
	}
	log.Debug("Token Challenge: Sending signature")
	err = websocket.JSON.Send(conn, solMessage)
	if err != nil {
		return nil, err
	}

	// Receive the token
	var token inMessageToken
	for token.Type != "token" {
		currToken := inMessageToken{}
		if err := websocket.JSON.Receive(conn, &token); err != nil {
			return nil, err
		}

		if currToken.Type == "token" {
			token = currToken
		}
	}
	if token.Type == "token" {
		log.Debug("Token Challenge: Received token successfully")
		return &token.Value, nil
	}

	return nil, errors.New("Did not receive a correct token challenge response")
}

func GetTokensWithCache(ctx context.Context, st store.Store, kc keychain.Keychain, cfg config.Config) (*AuthTokens, error) {
	if tokensInStore, _ := retrieveTokens(st); tokensInStore != nil {
		return &AuthTokens{
			HubToken: tokensInStore.Token,
			AppToken: tokensInStore.AppToken,
			Key:      tokensInStore.Key,
			Sig:      tokensInStore.Sig,
			Msg:      tokensInStore.Msg,
		}, nil
	}

	tokens, err := getTokensThroughChallenge(ctx, kc, cfg)
	if err != nil {
		return nil, err
	}

	if err := storeTokens(st, tokens); err != nil {
		return nil, err
	}

	return &AuthTokens{
		HubToken: tokens.Token,
		AppToken: tokens.AppToken,
		Key:      tokens.Key,
		Sig:      tokens.Sig,
		Msg:      tokens.Msg,
	}, nil
}

func GetHubContext(ctx context.Context, st store.Store, kc keychain.Keychain, cfg config.Config) (context.Context, error) {
	tokens, err := GetTokensWithCache(ctx, st, kc, cfg)
	if err != nil {
		return nil, err
	}

	_, sig, err := mbase.Decode(tokens.Sig)
	if err != nil {
		return nil, err
	}

	ctx = common.NewAPIKeyContext(ctx, tokens.Key)
	ctx = common.NewAPISigContext(ctx, tokens.Msg, sig)

	tok := thread.Token(tokens.HubToken)
	ctx = thread.NewTokenContext(ctx, tok)

	return ctx, nil
}
