package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"bansos-backend/app/model"
	"bansos-backend/app/repository"
	"bansos-backend/helper"
)

type ClaimService struct {
	pool    *pgxpool.Pool
	tx      *repository.TxManager
	claims  repository.ClaimRepository
	batches repository.BatchRepository
	perms   PermissionChecker
}

func NewClaimService(pool *pgxpool.Pool, claims repository.ClaimRepository, batches repository.BatchRepository, perms PermissionChecker) *ClaimService {
	return &ClaimService{
		pool:    pool,
		tx:      repository.NewTxManager(pool),
		claims:  claims,
		batches: batches,
		perms:   perms,
	}
}

func claimError(err error) error {
	var appErr *helper.AppError
	switch {
	case errors.As(err, &appErr):
		return appErr
	case errors.Is(err, repository.ErrClaimNotFound):
		return helper.NotFound("klaim tidak ditemukan")
	case errors.Is(err, repository.ErrBatchNotFound):
		return helper.NotFound("batch tidak ditemukan")
	}
	return helper.Internal(err)
}

func strPtr(s string) *string { return &s }

// ClaimSlot adalah alur war slot. Urutan kunci selalu batch dulu baru klaim,
// sama dengan alur ubah status, supaya tidak terjadi deadlock.
func (s *ClaimService) ClaimSlot(ctx context.Context, userID, batchID int) (*model.Claim, error) {
	b, err := s.batches.FindByID(ctx, s.pool, batchID)
	if err != nil {
		return nil, err
	}
	if !IsBatchVisible(b.Status, false) {
		return nil, repository.ErrBatchNotFound
	}
	if !IsBatchAcceptingClaims(b, time.Now()) {
		return nil, helper.Conflict("batch tidak sedang membuka war")
	}

	var created *model.Claim
	err = s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		if _, err := s.claims.FindApprovedByUserAndModule(ctx, tx, userID, b.ModuleID); err == nil {
			return helper.Conflict("kamu sudah punya klaim yang disetujui di modul ini")
		} else if !errors.Is(err, repository.ErrClaimNotFound) {
			return err
		}

		status := model.ClaimPending
		taken, err := s.batches.TakeSlotInWindow(ctx, tx, batchID)
		if err != nil {
			return err
		}
		if !taken {
			// Kunci batch lalu cek ulang: penyebabnya bisa kuota penuh,
			// bisa juga war baru saja ditutup.
			locked, err := s.batches.FindByIDForUpdate(ctx, tx, batchID)
			if err != nil {
				return err
			}
			if !IsBatchAcceptingClaims(locked, time.Now()) {
				return helper.Conflict("batch tidak sedang membuka war")
			}
			if taken, err = s.batches.TakeSlotInWindow(ctx, tx, batchID); err != nil {
				return err
			}
			if !taken {
				status = model.ClaimWaitlisted
			}
		}

		created, err = s.claims.Create(ctx, tx, userID, batchID, status)
		switch {
		case repository.IsUniqueViolation(err):
			return helper.Conflict("kamu sudah mengklaim batch ini")
		case repository.IsForeignKeyViolation(err):
			return helper.Unauthorized("akun tidak terdaftar")
		case err != nil:
			return err
		}

		_, err = s.claims.InsertLog(ctx, tx, model.ClaimLog{
			ClaimID: created.ID, ActorID: userID, ToStatus: status, Note: "klaim slot",
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// changeStatus dipakai pembatalan oleh peserta dan keputusan pengelola.
// check dijalankan setelah klaim terkunci, untuk aturan kepemilikan dan transisi.
func (s *ClaimService) changeStatus(ctx context.Context, actorID, claimID int, to, note string, check func(*model.Claim) error) (*model.Claim, error) {
	var updated *model.Claim
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		peek, err := s.claims.FindByID(ctx, tx, claimID)
		if err != nil {
			return err
		}
		if _, err := s.batches.FindByIDForUpdate(ctx, tx, peek.BatchID); err != nil {
			return err
		}
		current, err := s.claims.FindByIDForUpdate(ctx, tx, claimID)
		if err != nil {
			return err
		}
		if err := check(current); err != nil {
			return err
		}

		if TransitionTakesSlot(current.Status, to) {
			taken, err := s.batches.TakeSlot(ctx, tx, current.BatchID)
			if err != nil {
				return err
			}
			if !taken {
				return helper.Conflict("kuota batch sudah penuh")
			}
		}

		updated, err = s.claims.UpdateStatus(ctx, tx, claimID, to)
		if err != nil {
			return err
		}
		if _, err := s.claims.InsertLog(ctx, tx, model.ClaimLog{
			ClaimID: claimID, ActorID: actorID, FromStatus: strPtr(current.Status), ToStatus: to, Note: note,
		}); err != nil {
			return err
		}

		if TransitionReleasesSlot(current.Status, to) {
			if err := s.batches.ReleaseSlot(ctx, tx, current.BatchID); err != nil {
				return err
			}
			return s.promoteWaitlist(ctx, tx, actorID, current.BatchID, claimID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// promoteWaitlist mengisi slot yang baru kosong dengan antrean terdepan.
func (s *ClaimService) promoteWaitlist(ctx context.Context, tx repository.DBTX, actorID, batchID, excludeID int) error {
	next, err := s.claims.NextWaitlistedForUpdate(ctx, tx, batchID, excludeID)
	if errors.Is(err, repository.ErrClaimNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	taken, err := s.batches.TakeSlot(ctx, tx, batchID)
	if err != nil || !taken {
		return err
	}
	if _, err := s.claims.UpdateStatus(ctx, tx, next.ID, model.ClaimPending); err != nil {
		return err
	}
	_, err = s.claims.InsertLog(ctx, tx, model.ClaimLog{
		ClaimID: next.ID, ActorID: actorID, FromStatus: strPtr(model.ClaimWaitlisted),
		ToStatus: model.ClaimPending, Note: "dipromosikan otomatis dari daftar tunggu",
	})
	return err
}

func (s *ClaimService) CancelClaim(ctx context.Context, userID, claimID int, note string) (*model.Claim, error) {
	if note == "" {
		note = "dibatalkan peserta"
	}
	return s.changeStatus(ctx, userID, claimID, model.ClaimCancelled, note, func(c *model.Claim) error {
		return CheckUserCancel(c, userID)
	})
}

func (s *ClaimService) VerifyClaim(ctx context.Context, actorID, claimID int, to, note string) (*model.Claim, error) {
	return s.changeStatus(ctx, actorID, claimID, to, note, func(c *model.Claim) error {
		return CheckClaimTransition(c.Status, to)
	})
}

func (s *ClaimService) can(c *fiber.Ctx, permission string) bool {
	user, ok := helper.CurrentUser(c)
	return ok && s.perms != nil && s.perms.Can(user.Role, permission)
}

func mustUser(c *fiber.Ctx) (model.AuthUser, error) {
	user, ok := helper.CurrentUser(c)
	if !ok {
		return user, helper.Unauthorized("belum terautentikasi")
	}
	return user, nil
}

func (s *ClaimService) Create(c *fiber.Ctx) error {
	user, err := mustUser(c)
	if err != nil {
		return err
	}
	var req model.CreateClaimRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body JSON tidak valid")
	}
	if fields := helper.ValidateStruct(req); fields != nil {
		return helper.Validation(fields)
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	claim, err := s.ClaimSlot(ctx, user.UserID, req.BatchID)
	if err != nil {
		return claimError(err)
	}
	message := "slot berhasil diamankan, menunggu persetujuan pengelola"
	if claim.Status == model.ClaimWaitlisted {
		message = "kuota penuh, kamu masuk daftar tunggu"
	}
	return helper.Created(c, message, claim, "/api/v1/claims/"+strconv.Itoa(claim.ID))
}

func (s *ClaimService) Cancel(c *fiber.Ctx) error {
	user, err := mustUser(c)
	if err != nil {
		return err
	}
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id klaim tidak valid")
	}
	var req model.CancelClaimRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return helper.BadRequest("body JSON tidak valid")
		}
	}
	if fields := helper.ValidateStruct(req); fields != nil {
		return helper.Validation(fields)
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	claim, err := s.CancelClaim(ctx, user.UserID, id, req.Note)
	if err != nil {
		return claimError(err)
	}
	return helper.Success(c, fiber.StatusOK, "OK", "klaim dibatalkan", claim)
}

func (s *ClaimService) Verify(c *fiber.Ctx) error {
	user, err := mustUser(c)
	if err != nil {
		return err
	}
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id klaim tidak valid")
	}
	var req model.VerifyClaimRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body JSON tidak valid")
	}
	if fields := helper.ValidateStruct(req); fields != nil {
		return helper.Validation(fields)
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	claim, err := s.VerifyClaim(ctx, user.UserID, id, req.Status, req.Note)
	if err != nil {
		return claimError(err)
	}
	return helper.Success(c, fiber.StatusOK, "OK", "status klaim menjadi "+claim.Status, claim)
}

// findVisible mengembalikan klaim kalau milik sendiri atau pemanggil punya
// claim:read:any. Selain itu 404 supaya keberadaan klaim orang lain tidak bocor.
func (s *ClaimService) findVisible(c *fiber.Ctx, ctx context.Context) (*model.Claim, error) {
	user, err := mustUser(c)
	if err != nil {
		return nil, err
	}
	id, ok := helper.ParamID(c)
	if !ok {
		return nil, helper.BadRequest("id klaim tidak valid")
	}
	claim, err := s.claims.FindByID(ctx, s.pool, id)
	if err != nil {
		return nil, claimError(err)
	}
	if s.can(c, "claim:read:any") {
		return claim, nil
	}
	if err := CheckClaimOwnership(claim, user.UserID); err != nil {
		return nil, err
	}
	return claim, nil
}

func (s *ClaimService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	claim, err := s.findVisible(c, ctx)
	if err != nil {
		return err
	}
	return helper.Success(c, fiber.StatusOK, "OK", "detail klaim", claim)
}

func (s *ClaimService) Logs(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	claim, err := s.findVisible(c, ctx)
	if err != nil {
		return err
	}
	logs, err := s.claims.ListLogs(ctx, s.pool, claim.ID)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "OK", "log perubahan status klaim", logs)
}

func parseClaimQuery(c *fiber.Ctx) (model.ClaimListFilter, error) {
	var q model.ClaimHistoryQuery
	if err := c.QueryParser(&q); err != nil {
		return model.ClaimListFilter{}, helper.BadRequest("query parameter tidak valid")
	}
	if fields := helper.ValidateStruct(q); fields != nil {
		return model.ClaimListFilter{}, helper.Validation(fields)
	}
	cursor, err := helper.DecodeCursor(q.Cursor)
	if err != nil {
		return model.ClaimListFilter{}, err
	}
	return model.ClaimListFilter{Status: q.Status, Limit: helper.NormalizeLimit(q.Limit), Cursor: cursor}, nil
}

func (s *ClaimService) MyHistory(c *fiber.Ctx) error {
	user, err := mustUser(c)
	if err != nil {
		return err
	}
	filter, err := parseClaimQuery(c)
	if err != nil {
		return err
	}
	filter.UserID = user.UserID

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	rows, err := s.claims.ListByUser(ctx, s.pool, filter)
	if err != nil {
		return helper.Internal(err)
	}
	page, meta := helper.Paginate(rows, filter.Limit, func(d model.ClaimDetail) model.Cursor {
		return model.Cursor{CreatedAt: d.CreatedAt, ID: d.ID}
	})
	return helper.SuccessCursor(c, "riwayat klaim", page, &meta)
}

func (s *ClaimService) ListByBatch(c *fiber.Ctx) error {
	batchID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id batch tidak valid")
	}
	filter, err := parseClaimQuery(c)
	if err != nil {
		return err
	}
	filter.BatchID = batchID

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	if _, err := s.batches.FindByID(ctx, s.pool, batchID); err != nil {
		return claimError(err)
	}
	rows, err := s.claims.ListByBatch(ctx, s.pool, filter)
	if err != nil {
		return helper.Internal(err)
	}
	page, meta := helper.Paginate(rows, filter.Limit, func(cl model.Claim) model.Cursor {
		return model.Cursor{CreatedAt: cl.CreatedAt, ID: cl.ID}
	})
	return helper.SuccessCursor(c, "klaim pada batch", page, &meta)
}
