package service

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"bansos-backend/app/model"
	"bansos-backend/app/repository"
	"bansos-backend/helper"
)

type UserService struct {
	pool   *pgxpool.Pool
	tx     *repository.TxManager
	users  repository.UserRepository
	tokens repository.TokenRepository
}

func NewUserService(pool *pgxpool.Pool, users repository.UserRepository, tokens repository.TokenRepository) *UserService {
	return &UserService{
		pool:   pool,
		tx:     repository.NewTxManager(pool),
		users:  users,
		tokens: tokens,
	}
}

func userError(err error) error {
	var appErr *helper.AppError
	switch {
	case errors.As(err, &appErr):
		return appErr
	case errors.Is(err, repository.ErrUserNotFound):
		return helper.NotFound("pengguna tidak ditemukan")
	}
	return helper.Internal(err)
}

func (s *UserService) List(c *fiber.Ctx) error {
	var q model.UserListQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.BadRequest("query parameter tidak valid")
	}
	if fields := helper.ValidateStruct(q); fields != nil {
		return helper.Validation(fields)
	}

	cursor, err := helper.DecodeCursor(q.Cursor)
	if err != nil {
		return err
	}
	limit := helper.NormalizeLimit(q.Limit)

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	rows, err := s.users.List(ctx, s.pool, model.UserListFilter{
		Role:   q.Role,
		Search: strings.TrimSpace(q.Search),
		Limit:  limit,
		Cursor: cursor,
	})
	if err != nil {
		return helper.Internal(err)
	}

	page, meta := helper.Paginate(rows, limit, func(u model.User) model.Cursor {
		return model.Cursor{CreatedAt: u.CreatedAt, ID: u.ID}
	})
	return helper.SuccessCursor(c, "daftar pengguna", page, &meta)
}

func (s *UserService) Get(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id pengguna tidak valid")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, err := s.users.FindByID(ctx, s.pool, id)
	if err != nil {
		return userError(err)
	}
	return helper.Success(c, fiber.StatusOK, "OK", "detail pengguna", user)
}

// UpdateRole mengubah role pengguna. Semua refresh token pengguna dicabut supaya
// sesi lama tidak terus memakai permission role sebelumnya.
func (s *UserService) UpdateRole(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id pengguna tidak valid")
	}

	var req model.UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body JSON tidak valid")
	}
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	if fields := ValidateUpdateRole(req); len(fields) > 0 {
		return helper.Validation(fields)
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var updated *model.User
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		var err error
		updated, err = s.users.UpdateRole(ctx, tx, id, req.Role)
		if err != nil {
			return err
		}
		return s.tokens.RevokeAllForUser(ctx, tx, id)
	})
	if err != nil {
		return userError(err)
	}
	return helper.Success(c, fiber.StatusOK, "OK", "role pengguna menjadi "+updated.Role, updated)
}
