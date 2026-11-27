package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"bansos-backend/app/model"
	"bansos-backend/app/repository"
	"bansos-backend/helper"
)

type PermissionChecker interface {
	Can(role, permission string) bool
}

type ModuleService struct {
	pool  *pgxpool.Pool
	tx    *repository.TxManager
	repo  repository.ModuleRepository
	perms PermissionChecker
}

func NewModuleService(pool *pgxpool.Pool, repo repository.ModuleRepository, perms PermissionChecker) *ModuleService {
	return &ModuleService{
		pool:  pool,
		tx:    repository.NewTxManager(pool),
		repo:  repo,
		perms: perms,
	}
}

func (s *ModuleService) canManage(c *fiber.Ctx) bool {
	user, ok := helper.CurrentUser(c)
	return ok && s.perms != nil && s.perms.Can(user.Role, "module:manage")
}

func (s *ModuleService) List(c *fiber.Ctx) error {
	var q model.ModuleListQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.BadRequest("query parameter tidak valid")
	}
	if fields := helper.ValidateStruct(q); fields != nil {
		return helper.Validation(fields)
	}

	status, ok := ResolveModuleStatus(q.Status, s.canManage(c))
	if !ok {
		return helper.Forbidden("hanya pengelola modul yang boleh melihat modul nonaktif")
	}

	cursor, err := helper.DecodeCursor(q.Cursor)
	if err != nil {
		return err
	}

	limit := helper.NormalizeLimit(q.Limit)
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	rows, err := s.repo.List(ctx, s.pool, model.ModuleListFilter{
		Category: NormalizeCategory(q.Category),
		Level:    q.Level,
		Status:   status,
		Limit:    limit,
		Cursor:   cursor,
	})
	if err != nil {
		return helper.Internal(err)
	}

	page, meta := helper.Paginate(rows, limit, func(m model.Module) model.Cursor {
		return model.Cursor{CreatedAt: m.CreatedAt, ID: m.ID}
	})
	return helper.SuccessCursor(c, "katalog modul", page, &meta)
}

func (s *ModuleService) Get(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id modul tidak valid")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	m, err := s.repo.FindByID(ctx, s.pool, id)
	if errors.Is(err, repository.ErrModuleNotFound) || (err == nil && !m.IsActive && !s.canManage(c)) {
		return helper.NotFound("modul tidak ditemukan")
	}
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "OK", "detail modul", m)
}

func (s *ModuleService) Create(c *fiber.Ctx) error {
	var req model.CreateModuleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body JSON tidak valid")
	}
	NormalizeCreateModule(&req)
	if fields := helper.ValidateStruct(req); fields != nil {
		return helper.Validation(fields)
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	m, err := s.repo.Create(ctx, s.pool, req)
	if repository.IsUniqueViolation(err) {
		return helper.Conflict("kode modul " + req.Code + " sudah dipakai")
	}
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Created(c, "modul berhasil dibuat", m, "/api/v1/modules/"+strconv.Itoa(m.ID))
}

func (s *ModuleService) Update(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id modul tidak valid")
	}

	var req model.UpdateModuleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body JSON tidak valid")
	}
	NormalizeUpdateModule(&req)
	if fields := helper.ValidateStruct(req); fields != nil {
		return helper.Validation(fields)
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var updated *model.Module
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		current, err := s.repo.FindByIDForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		if current.IsActive && !*req.IsActive {
			if err := s.ensureNoRunningBatch(ctx, tx, id); err != nil {
				return err
			}
		}
		updated, err = s.repo.Update(ctx, tx, id, req)
		return err
	})
	switch {
	case err == nil:
		return helper.Success(c, fiber.StatusOK, "OK", "modul berhasil diperbarui", updated)
	case errors.Is(err, repository.ErrModuleNotFound):
		return helper.NotFound("modul tidak ditemukan")
	case repository.IsUniqueViolation(err):
		return helper.Conflict("kode modul " + req.Code + " sudah dipakai")
	}
	var appErr *helper.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return helper.Internal(err)
}

// Delete menonaktifkan modul (soft delete) karena batch, klaim, dan sertifikat
// lama masih merujuk ke modul ini.
func (s *ModuleService) Delete(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id modul tidak valid")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		current, err := s.repo.FindByIDForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		if !current.IsActive {
			return nil
		}
		if err := s.ensureNoRunningBatch(ctx, tx, id); err != nil {
			return err
		}
		return s.repo.Deactivate(ctx, tx, id)
	})
	if errors.Is(err, repository.ErrModuleNotFound) {
		return helper.NotFound("modul tidak ditemukan")
	}
	var appErr *helper.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	if err != nil {
		return helper.Internal(err)
	}
	return helper.NoContent(c)
}

func (s *ModuleService) ensureNoRunningBatch(ctx context.Context, tx repository.DBTX, moduleID int) error {
	n, err := s.repo.CountRunningBatches(ctx, tx, moduleID)
	if err != nil {
		return err
	}
	if n > 0 {
		return helper.Conflict("modul masih punya " + strconv.Itoa(n) + " batch terjadwal atau sedang buka")
	}
	return nil
}
