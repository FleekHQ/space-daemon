package hub

import (
	"encoding/base32"
	"strings"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/libp2p/go-libp2p-core/crypto"
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

const base32Alphabet = "abcdefghijklmnopqrstuvwxyz234567"
const hubTokenStoreKey = "hubAuthToken"

var lowerBase32 = base32.NewEncoding(base32Alphabet)

func encodeToString(in []byte) string {
	return strings.TrimRight(lowerBase32.EncodeToString(in), "=")
}

func getHubTokenFromStore(st store.Store) (string, error) {
	key := []byte(hubTokenStoreKey)
	val, _ := st.Get(key)

	if val == nil {
		return "", nil
	}

	return string(val), nil
}

func storeHubTokenToStore(st store.Store, hubToken string) error {
	err := st.Set([]byte(hubTokenStoreKey), []byte(hubToken))

	return err
}

func GetHubToken(store store.Store, cfg config.Config) (string, error) {
	// Try to avoid redoing challenge if we already have the token
	if valFromStore, err := getHubTokenFromStore(store); err != nil {
		return "", err
	} else if valFromStore != "" {
		log.Debug("Token Challenge: Got token from store: " + valFromStore)
		return valFromStore, nil
	}

	kc := keychain.New(store)
	log.Debug("Token Challenge: Connecting through websocket")
	conn, err := websocket.Dial(cfg.GetString(config.SpaceServicesHubAuthURL, ""), "", "http://localhost/")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	log.Debug("Token Challenge: Connected")

	_, pub, err := kc.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return "", err
	}

	publicKeyBytes, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		return "", err
	}

	// Textile auth requires public key to be in base32 format with a particular alphabet
	publicKeyBase32 := encodeToString(publicKeyBytes)

	// Request a challenge (a payload we need to sign)
	log.Debug("Token Challenge: Sending token request with pub key", publicKeyBase32)
	tokenRequest := &outMessage{
		Action: "token",
		Data: sentMessageData{
			PublicKey: publicKeyBase32,
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

	solution, err := kc.Sign(challenge.Value.Data)
	if err != nil {
		return "", err
	}

	// Send back channel solution
	solMessage := &outMessage{
		Action: "challenge",
		Data: sentMessageData{
			Signature: solution,
			PublicKey: publicKeyBase32,
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

	if err := storeHubTokenToStore(store, token.Value.Token); err != nil {
		return "", err
	}

	return token.Value.Token, nil
}
