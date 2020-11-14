package app_token_auth

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/permissions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AppTokenAuth struct {
	kc keychain.Keychain
}

func New(kc keychain.Keychain) *AppTokenAuth {
	return &AppTokenAuth{
		kc: kc,
	}
}

func (a *AppTokenAuth) Authorize(ctx context.Context, fullMethodName string) (context.Context, error) {
	if canSkipAuth(fullMethodName) {
		return ctx, nil
	}

	token, err := AuthFromMD(ctx, "AppToken")
	if err != nil {
		return nil, err
	}

	tokenInfo, err := a.validateToken(token, fullMethodName)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	newCtx := context.WithValue(ctx, "appToken", tokenInfo)

	return newCtx, nil
}

func (a *AppTokenAuth) validateToken(tok, fullMethodName string) (*permissions.AppToken, error) {
	key, sec, err := permissions.GetKeyAndSecretFromAccessToken(tok)
	if err != nil {
		return nil, err
	}

	appTok, err := a.kc.GetAppToken(key)
	if err != nil {
		return nil, err
	}

	if appTok.Secret != sec {
		return nil, errors.New("app token secret does not match")
	}

	authorized := false

	if appTok.IsMaster {
		authorized = true
	}

	// Check if method is authorized
	for _, p := range appTok.Permissions {
		if "/space.SpaceApi/"+p == fullMethodName {
			authorized = true
		}
	}

	if authorized == false {
		return nil, errors.New("app token does not grant access to " + fullMethodName)
	}

	return appTok, nil
}

var publicMethods = []string{
	"InitializeMasterAppToken",
}

func canSkipAuth(fullMethodName string) bool {
	for _, pm := range publicMethods {
		if "/space.SpaceApi/"+pm == fullMethodName {
			return true
		}
	}

	return false
}
