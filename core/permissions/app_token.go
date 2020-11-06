package permissions

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
)

var invalidAppTokenErr = errors.New("app token is invalid")

const tokenKeyLength = 20
const tokenSecretLength = 30

type AppToken struct {
	Key         string
	Secret      string
	IsMaster    bool
	Permissions []string
}

// Token structure is [KEY].[SECRET].[IS_ADMIN].[PERMISSION1]_[PERMISSION2]...
func UnmarshalFullToken(fullTok string) (*AppToken, error) {
	parsed := strings.Split(fullTok, ".")
	if len(parsed) < 3 {
		return nil, invalidAppTokenErr
	}

	isMaster := parsed[2] == "true"

	permissions := make([]string, 0)

	if len(parsed) >= 4 {
		permissions = strings.Split(parsed[3], "_")
	}

	return &AppToken{
		Key:         parsed[0],
		Secret:      parsed[1],
		IsMaster:    isMaster,
		Permissions: permissions,
	}, nil
}

func MarshalFullToken(tok *AppToken) string {
	isMaster := "false"

	if tok.IsMaster {
		isMaster = "true"
	}

	return strings.Join([]string{tok.Key, tok.Secret, isMaster, strings.Join(tok.Permissions, "_")}, ".")
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
