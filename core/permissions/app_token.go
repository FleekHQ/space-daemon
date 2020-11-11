package permissions

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

var invalidAppTokenErr = errors.New("app token is invalid")

const tokenKeyLength = 20
const tokenSecretLength = 30

type AppToken struct {
	Key         string   `json:"key"`
	Secret      string   `json:"secret"`
	IsMaster    bool     `json:"isMaster"`
	Permissions []string `json:"permissions"`
}

func UnmarshalToken(marshalledToken []byte) (*AppToken, error) {
	var result AppToken
	err := json.Unmarshal(marshalledToken, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func MarshalToken(tok *AppToken) ([]byte, error) {
	jsonData, err := json.Marshal(tok)
	if err != nil {
		return nil, err
	}

	return jsonData, nil

}

func GenerateRandomToken(isMaster bool, permissions []string) (*AppToken, error) {
	k := make([]byte, tokenKeyLength)
	_, err := rand.Read(k)
	if err != nil {
		return nil, err
	}

	s := make([]byte, tokenSecretLength)
	_, err = rand.Read(s)
	if err != nil {
		return nil, err
	}

	return &AppToken{
		Key:         base64.RawURLEncoding.EncodeToString(k),
		Secret:      base64.RawURLEncoding.EncodeToString(s),
		IsMaster:    isMaster,
		Permissions: permissions,
	}, nil
}

func (a *AppToken) GetAccessToken() string {
	return a.Key + "." + a.Secret
}

func GetKeyAndSecretFromAccessToken(accessToken string) (key string, secret string, err error) {
	tp := strings.Split(accessToken, ".")
	if len(tp) < 2 {
		return "", "", errors.New("invalid token format")
	}

	key = tp[0]
	secret = tp[1]

	return
}
