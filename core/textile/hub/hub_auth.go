package hub

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/dgrijalva/jwt-go"
	mbase "github.com/multiformats/go-multibase"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/textile/v2/api/common"
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

type HubAuth interface {
	GetTokensWithCache(ctx context.Context) (*AuthTokens, error)
	GetHubContext(ctx context.Context) (context.Context, error)
	ClearCache() error
}

type hub struct {
	st               store.Store
	kc               keychain.Keychain
	cfg              config.Config
	fetchTokensMutex *sync.Mutex
}

func New(st store.Store, kc keychain.Keychain, cfg config.Config) *hub {
	return &hub{
		st:               st,
		kc:               kc,
		cfg:              cfg,
		fetchTokensMutex: &sync.Mutex{},
	}
}

const tokensStoreKey = "hubTokens"

func isTokenExpired(t string) bool {
	token, _, err := new(jwt.Parser).ParseUnverified(t, jwt.MapClaims{})
	if err != nil {
		return true
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return true
	}

	var expiryTime time.Time
	switch exp := claims["exp"].(type) {
	case float64:
		expiryTime = time.Unix(int64(exp), 0)
	case json.Number:
		v, err := exp.Int64()
		if err != nil {
			return true
		}
		expiryTime = time.Unix(v, 0)
	}

	now := time.Now()

	return expiryTime.Before(now)
}

func (h *hub) retrieveTokens() (*inMessageTokenValue, error) {
	stored, err := h.st.Get([]byte(tokensStoreKey))
	if err != nil {
		return nil, err
	}

	tokens := &inMessageTokenValue{}
	tokensBytes := bytes.NewBuffer(stored)

	if err := json.NewDecoder(tokensBytes).Decode(tokens); err != nil {
		return nil, err
	}

	expired := isTokenExpired(tokens.AppToken)
	if expired {
		return nil, errors.New("App token is expired")
	}

	return tokens, nil
}

func (h *hub) storeTokens(tokens *inMessageTokenValue) error {
	tokensBytes := new(bytes.Buffer)
	if err := json.NewEncoder(tokensBytes).Encode(tokens); err != nil {
		return err
	}

	if err := h.st.Set([]byte(tokensStoreKey), tokensBytes.Bytes()); err != nil {
		return err
	}

	return nil
}

// Removes the stored tokens
func (h *hub) ClearCache() error {
	return h.st.Remove([]byte(tokensStoreKey))
}

func (h *hub) getTokensThroughChallenge(ctx context.Context) (*inMessageTokenValue, error) {
	log.Debug("Token Challenge: Connecting through websocket")
	conn, err := websocket.Dial(h.cfg.GetString(config.SpaceServicesHubAuthURL, ""), "", "http://localhost/")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	log.Debug("Token Challenge: Connected")

	privateKey, _, err := h.kc.GetStoredKeyPairInLibP2PFormat()

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

func (h *hub) GetTokensWithCache(ctx context.Context) (*AuthTokens, error) {
	h.fetchTokensMutex.Lock()
	defer h.fetchTokensMutex.Unlock()
	if tokensInStore, _ := h.retrieveTokens(); tokensInStore != nil {
		return &AuthTokens{
			HubToken: tokensInStore.Token,
			AppToken: tokensInStore.AppToken,
			Key:      tokensInStore.Key,
			Sig:      tokensInStore.Sig,
			Msg:      tokensInStore.Msg,
		}, nil
	}

	tokens, err := h.getTokensThroughChallenge(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.storeTokens(tokens); err != nil {
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

func (h *hub) GetHubContext(ctx context.Context) (context.Context, error) {
	tokens, err := h.GetTokensWithCache(ctx)
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
