package service

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"bansos-backend/app/model"
	"bansos-backend/app/repository"
	"bansos-backend/helper"
)

type BatchService struct {
	pool    *pgxpool.Pool
	tx      *repository.TxManager
	repo    repository.BatchRepository
	modules repository.ModuleRepository
	perms   PermissionChecker
}

func NewBatchService(pool *pgxpool.Pool, repo repository.BatchRepository, modules repository.ModuleRepository, perms PermissionChecker) *BatchService {
	return &BatchService{
		pool:    pool,
		tx:      repository.NewTxManager(pool),
		repo:    repo,
		modules: modules,
		perms:   perms,
	}
}

func (s *BatchService) can(c *fiber.Ctx, permission string) bool {
	user, ok := helper.CurrentUser(c)
	return ok && s.perms != nil && s.perms.Can(user.Role, permission)
}

func batchError(err error) error {
	var appErr *helper.AppError
	switch {
	case errors.As(err, &appErr):
		return appErr
	case errors.Is(err, repository.ErrBatchNotFound):
		return helper.NotFound("batch tidak ditemukan")
	case errors.Is(err, repository.ErrModuleNotFound):
		return helper.NotFound("modul tidak ditemukan")
	}
	return helper.Internal(err)
}

func (s *BatchService) ListByModule(c *fiber.Ctx) error {
	moduleID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id modul tidak valid")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	m, err := s.modules.FindByID(ctx, s.pool, moduleID)
	if err != nil {
		return batchError(err)
	}
	if !m.IsActive && !s.can(c, "module:manage") {
		return helper.NotFound("modul tidak ditemukan")
	}

	batches, err := s.repo.ListByModule(ctx, s.pool, moduleID, s.can(c, "batch:manage"))
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "OK", "daftar batch modul "+m.Code, batches)
}

func (s *BatchService) Get(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id batch tidak valid")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	b, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return batchError(err)
	}
	if !IsBatchVisible(b.Status, s.can(c, "batch:manage")) {
		return helper.NotFound("batch tidak ditemukan")
	}
	return helper.Success(c, fiber.StatusOK, "OK", "detail batch", b)
}

func (s *BatchService) Create(c *fiber.Ctx) error {
	moduleID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id modul tidak valid")
	}

	var req model.CreateBatchRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body JSON tidak valid")
	}
	if fields := helper.ValidateStruct(req); fields != nil {
		return helper.Validation(fields)
	}
	if err := CheckBatchWindow(req.OpensAt, req.ClosesAt, time.Now()); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var created *model.Batch
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		m, err := s.modules.FindByIDForShare(ctx, tx, moduleID)
		if err != nil {
			return err
		}
		if !m.IsActive {
			return helper.Conflict("modul nonaktif, aktifkan dulu sebelum membuat batch")
		}
		created, err = s.repo.Create(ctx, tx, moduleID, req)
		return err
	})
	if err != nil {
		return batchError(err)
	}
	return helper.Created(c, "batch berhasil dibuat sebagai draft", created, "/api/v1/batches/"+strconv.Itoa(created.ID))
}

func (s *BatchService) Update(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id batch tidak valid")
	}

	var req model.UpdateBatchRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body JSON tidak valid")
	}
	if fields := helper.ValidateStruct(req); fields != nil {
		return helper.Validation(fields)
	}
	if err := CheckBatchWindow(req.OpensAt, req.ClosesAt, time.Now()); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var updated *model.Batch
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		current, err := s.repo.FindByIDForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		if !IsBatchEditable(current.Status) {
			return helper.Conflict("batch berstatus " + current.Status + " tidak bisa diubah")
		}
		if err := CheckBatchQuota(req.Quota, current.Allocated); err != nil {
			return err
		}
		if current.Status == model.BatchScheduled && !req.OpensAt.After(time.Now()) {
			return helper.Validation(map[string]string{"opens_at": "batch terjadwal harus punya opens_at di masa depan"})
		}
		updated, err = s.repo.Update(ctx, tx, id, req)
		return err
	})
	if err != nil {
		return batchError(err)
	}
	return helper.Success(c, fiber.StatusOK, "OK", "batch berhasil diperbarui", updated)
}

func (s *BatchService) ChangeStatus(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id batch tidak valid")
	}

	var req model.ChangeBatchStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body JSON tidak valid")
	}
	if fields := helper.ValidateStruct(req); fields != nil {
		return helper.Validation(fields)
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var updated *model.Batch
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		current, err := s.repo.FindByIDForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := CheckBatchTransition(current, req.Status, time.Now()); err != nil {
			return err
		}
		if req.Status == model.BatchScheduled || req.Status == model.BatchOpen {
			m, err := s.modules.FindByIDForShare(ctx, tx, current.ModuleID)
			if err != nil {
				return err
			}
			if !m.IsActive {
				return helper.Conflict("modul nonaktif, batch tidak bisa dijadwalkan atau dibuka")
			}
		}
		updated, err = s.repo.UpdateStatus(ctx, tx, id, req.Status)
		return err
	})
	if err != nil {
		return batchError(err)
	}
	return helper.Success(c, fiber.StatusOK, "OK", "status batch menjadi "+updated.Status, updated)
}

// Delete hanya untuk batch draft. Batch yang pernah dijadwalkan cukup ditutup
// supaya riwayat klaim tetap utuh.
func (s *BatchService) Delete(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id batch tidak valid")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		current, err := s.repo.FindByIDForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		if current.Status != model.BatchDraft {
			return helper.Conflict("hanya batch draft yang bisa dihapus, batch lain cukup ditutup")
		}
		return s.repo.Delete(ctx, tx, id)
	})
	if err != nil {
		return batchError(err)
	}
	return helper.NoContent(c)
}
