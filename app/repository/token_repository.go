package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"bansos-backend/app/model"
)

var ErrTokenNotFound = errors.New("refresh token tidak ditemukan")

const refreshTokenColumns = `rt.id, rt.user_id, rt.token_hash, rt.expires_at, rt.revoked_at, rt.created_at`

func scanRefreshToken(row pgx.Row) (*model.RefreshToken, error) {
	var t model.RefreshToken
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type TokenRepository interface {
	Save(ctx context.Context, db DBTX, token model.RefreshToken) (*model.RefreshToken, error)
	FindActive(ctx context.Context, db DBTX, tokenHash string) (*model.RefreshToken, error)
	Revoke(ctx context.Context, db DBTX, tokenHash string) error
	RevokeAllForUser(ctx context.Context, db DBTX, userID int) error
}

type tokenRepository struct{}

func NewTokenRepository() TokenRepository {
	return &tokenRepository{}
}

func (r *tokenRepository) Save(ctx context.Context, db DBTX, token model.RefreshToken) (*model.RefreshToken, error) {
	row := db.QueryRow(ctx, `
		INSERT INTO refresh_tokens AS rt (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING `+refreshTokenColumns,
		token.UserID, token.TokenHash, token.ExpiresAt)
	return scanRefreshToken(row)
}

func (r *tokenRepository) FindActive(ctx context.Context, db DBTX, tokenHash string) (*model.RefreshToken, error) {
	row := db.QueryRow(ctx, `
		SELECT `+refreshTokenColumns+`
		FROM refresh_tokens rt
		WHERE rt.token_hash = $1
		  AND rt.revoked_at IS NULL
		  AND rt.expires_at > now()`, tokenHash)
	return scanRefreshToken(row)
}

func (r *tokenRepository) Revoke(ctx context.Context, db DBTX, tokenHash string) error {
	tag, err := db.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash)
	if err != nil {
		return fmt.Errorf("mencabut refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTokenNotFound
	}
	return nil
}

func (r *tokenRepository) RevokeAllForUser(ctx context.Context, db DBTX, userID int) error {
	_, err := db.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		return fmt.Errorf("mencabut seluruh refresh token pengguna: %w", err)
	}
	return nil
}
