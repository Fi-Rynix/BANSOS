package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"bansos-backend/app/model"
	"bansos-backend/app/repository"
	"bansos-backend/helper"
)

const refreshTokenBytes = 32

type AuthService struct {
	pool       *pgxpool.Pool
	tx         *repository.TxManager
	users      repository.UserRepository
	tokens     repository.TokenRepository
	jwt        *helper.JWTManager
	refreshTTL time.Duration
}

func NewAuthService(
	pool *pgxpool.Pool,
	users repository.UserRepository,
	tokens repository.TokenRepository,
	jwtManager *helper.JWTManager,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		pool:       pool,
		tx:         repository.NewTxManager(pool),
		users:      users,
		tokens:     tokens,
		jwt:        jwtManager,
		refreshTTL: refreshTTL,
	}
}

func authError(err error) error {
	var appErr *helper.AppError
	switch {
	case errors.As(err, &appErr):
		return appErr
	case errors.Is(err, repository.ErrUserNotFound), errors.Is(err, repository.ErrTokenNotFound):
		return helper.Unauthorized("email atau password salah")
	case errors.Is(err, repository.ErrEmailTaken):
		return helper.Conflict("email sudah dipakai")
	}
	return helper.Internal(err)
}

// Register mendaftarkan akun baru. Role selalu peserta, hanya admin yang bisa
// menaikkan role lewat endpoint kelola pengguna.
func (s *AuthService) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body JSON tidak valid")
	}
	if fields := ValidateRegister(req); len(fields) > 0 {
		return helper.Validation(fields)
	}

	hashed, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Internal(err)
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	created, err := s.users.Create(ctx, s.pool, model.User{
		Name:     strings.TrimSpace(req.Name),
		Email:    NormalizeEmail(req.Email),
		Password: hashed,
		Role:     model.RolePeserta,
		IsActive: true,
	})
	if err != nil {
		if repository.IsUniqueViolation(err) {
			return helper.Conflict("email sudah dipakai")
		}
		return authError(err)
	}
	return helper.Created(c, "pendaftaran berhasil", created, "/api/v1/admin/users/"+strconv.Itoa(created.ID))
}

func (s *AuthService) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body JSON tidak valid")
	}
	if fields := ValidateLogin(req); len(fields) > 0 {
		return helper.Validation(fields)
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, err := s.users.FindByEmail(ctx, s.pool, NormalizeEmail(req.Email))
	if err != nil {
		helper.VerifyDummyPassword(req.Password)
		return helper.Unauthorized("email atau password salah")
	}

	if !helper.VerifyPassword(user.Password, req.Password) {
		return helper.Unauthorized("email atau password salah")
	}
	if !user.IsActive {
		return helper.Forbidden("akun dinonaktifkan")
	}

	pair, err := s.issueTokenPair(ctx, s.pool, user)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "OK", "login berhasil", pair)
}

// Refresh merotasi refresh token: token lama dicabut, pasangan baru diterbitkan
// di dalam satu transaksi supaya kegagalan penerbitan tidak mematikan sesi.
func (s *AuthService) Refresh(c *fiber.Ctx) error {
	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body JSON tidak valid")
	}

	raw := strings.TrimSpace(req.RefreshToken)
	if raw == "" {
		return helper.BadRequest("refresh_token wajib diisi")
	}
	hash := helper.SHA256Hex(raw)

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var pair model.TokenPair
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		stored, err := s.tokens.FindActive(ctx, tx, hash)
		if err != nil {
			return helper.Unauthorized("refresh token tidak valid atau sudah kedaluwarsa")
		}

		user, err := s.users.FindByID(ctx, tx, stored.UserID)
		if err != nil {
			return helper.Unauthorized("akun tidak dapat dipakai")
		}
		if !user.IsActive {
			return helper.Forbidden("akun dinonaktifkan")
		}

		if err := s.tokens.Revoke(ctx, tx, hash); err != nil {
			return err
		}
		pair, err = s.issueTokenPair(ctx, tx, user)
		return err
	})
	if err != nil {
		return authError(err)
	}
	return helper.Success(c, fiber.StatusOK, "OK", "token berhasil diperbarui", pair)
}

// Logout mencabut refresh token. Access token yang sudah terbit tetap berlaku
// sampai masa berlakunya habis.
func (s *AuthService) Logout(c *fiber.Ctx) error {
	var req model.RefreshRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return helper.BadRequest("body JSON tidak valid")
		}
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	if raw := strings.TrimSpace(req.RefreshToken); raw != "" {
		if err := s.tokens.Revoke(ctx, s.pool, helper.SHA256Hex(raw)); err != nil &&
			!errors.Is(err, repository.ErrTokenNotFound) {
			return helper.Internal(err)
		}
	}
	return helper.Success(c, fiber.StatusOK, "OK", "logout berhasil", nil)
}

func (s *AuthService) Me(c *fiber.Ctx) error {
	authUser, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, err := s.users.FindByID(ctx, s.pool, authUser.UserID)
	if err != nil {
		return helper.Unauthorized("pengguna dari token tidak ditemukan")
	}
	return helper.Success(c, fiber.StatusOK, "OK", "profil pengguna", user)
}

func (s *AuthService) issueTokenPair(ctx context.Context, db repository.DBTX, user *model.User) (model.TokenPair, error) {
	access, err := s.jwt.GenerateAccess(*user)
	if err != nil {
		return model.TokenPair{}, fmt.Errorf("membuat access token: %w", err)
	}

	refresh, err := helper.RandomToken(refreshTokenBytes)
	if err != nil {
		return model.TokenPair{}, err
	}

	expiresAt := time.Now().Add(s.refreshTTL)
	if _, err := s.tokens.Save(ctx, db, model.RefreshToken{
		UserID:    user.ID,
		TokenHash: helper.SHA256Hex(refresh),
		ExpiresAt: expiresAt,
	}); err != nil {
		return model.TokenPair{}, err
	}

	return model.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.jwt.AccessTTL().Seconds()),
	}, nil
}
