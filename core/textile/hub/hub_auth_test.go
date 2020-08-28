package hub

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func TestHubAuth_isTokenExpiredTrue(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().AddDate(0, 0, -1).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenStr, _ := token.SignedString(HmacTestKey)

	exp := isTokenExpired(tokenStr)
	assert.Equal(t, true, exp)
}

func TestHubAuth_isTokenExpiredFalse(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().AddDate(0, 0, 1).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenStr, _ := token.SignedString(HmacTestKey)

	exp := isTokenExpired(tokenStr)
	assert.Equal(t, false, exp)
}
