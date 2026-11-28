package middleware

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"bansos-backend/app/model"
)

const stubTokenPrefix = "stub."

var ErrStubToken = errors.New("stub token dibuang atau tidak sah")

func parseStubToken(token string) (model.AuthUser, error) {
	if !strings.HasPrefix(token, stubTokenPrefix) {
		return model.AuthUser{}, ErrStubToken
	}

	payload := strings.TrimPrefix(token, stubTokenPrefix)
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return model.AuthUser{}, ErrMalformed
	}

	var user model.AuthUser
	if err := json.Unmarshal(raw, &user); err != nil {
		return model.AuthUser{}, ErrMalformed
	}
	if user.UserID < 1 || user.Role == "" {
		return model.AuthUser{}, ErrMalformed
	}
	return user, nil
}

func StubToken(user model.AuthUser) string {
	raw, err := json.Marshal(user)
	if err != nil {
		return ""
	}
	return stubTokenPrefix + base64.RawURLEncoding.EncodeToString(raw)
}
