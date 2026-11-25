package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"bansos-backend/app/model"
)

var ErrClaimNotFound = errors.New("klaim tidak ditemukan")

type ClaimRepository interface {
	Create(ctx context.Context, db DBTX, userID, batchID int, status string) (*model.Claim, error)
	FindByID(ctx context.Context, db DBTX, id int) (*model.Claim, error)
	FindByIDForUpdate(ctx context.Context, db DBTX, id int) (*model.Claim, error)
	FindByUserAndBatch(ctx context.Context, db DBTX, userID, batchID int) (*model.Claim, error)
	FindApprovedByUserAndModule(ctx context.Context, db DBTX, userID, moduleID int) (*model.Claim, error)
	NextWaitlistedForUpdate(ctx context.Context, db DBTX, batchID int) (*model.Claim, error)
	UpdateStatus(ctx context.Context, db DBTX, id int, status string) (*model.Claim, error)
	ListByUser(ctx context.Context, db DBTX, filter model.ClaimListFilter) ([]model.ClaimDetail, error)
	ListByBatch(ctx context.Context, db DBTX, filter model.ClaimListFilter) ([]model.Claim, error)
	InsertLog(ctx context.Context, db DBTX, log model.ClaimLog) (*model.ClaimLog, error)
	ListLogs(ctx context.Context, db DBTX, claimID int) ([]model.ClaimLog, error)
}

type claimRepository struct{}

func NewClaimRepository() ClaimRepository {
	return &claimRepository{}
}

const claimColumns = `c.id, c.user_id, c.batch_id, c.status, c.claimed_at, c.decided_at, c.created_at`

func scanClaim(row pgx.Row) (*model.Claim, error) {
	var c model.Claim
	err := row.Scan(&c.ID, &c.UserID, &c.BatchID, &c.Status, &c.ClaimedAt, &c.DecidedAt, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrClaimNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *claimRepository) Create(ctx context.Context, db DBTX, userID, batchID int, status string) (*model.Claim, error) {
	row := db.QueryRow(ctx, `
		INSERT INTO claims AS c (user_id, batch_id, status)
		VALUES ($1, $2, $3)
		RETURNING `+claimColumns,
		userID, batchID, status)
	return scanClaim(row)
}

func (r *claimRepository) FindByID(ctx context.Context, db DBTX, id int) (*model.Claim, error) {
	row := db.QueryRow(ctx, `SELECT `+claimColumns+` FROM claims c WHERE c.id = $1`, id)
	return scanClaim(row)
}

func (r *claimRepository) FindByIDForUpdate(ctx context.Context, db DBTX, id int) (*model.Claim, error) {
	row := db.QueryRow(ctx, `SELECT `+claimColumns+` FROM claims c WHERE c.id = $1 FOR UPDATE`, id)
	return scanClaim(row)
}

func (r *claimRepository) FindByUserAndBatch(ctx context.Context, db DBTX, userID, batchID int) (*model.Claim, error) {
	row := db.QueryRow(ctx, `
		SELECT `+claimColumns+`
		FROM claims c
		WHERE c.user_id = $1 AND c.batch_id = $2`,
		userID, batchID)
	return scanClaim(row)
}

func (r *claimRepository) FindApprovedByUserAndModule(ctx context.Context, db DBTX, userID, moduleID int) (*model.Claim, error) {
	row := db.QueryRow(ctx, `
		SELECT `+claimColumns+`
		FROM claims c
		JOIN batches b ON b.id = c.batch_id
		WHERE c.user_id = $1 AND b.module_id = $2 AND c.status = $3
		ORDER BY c.decided_at DESC NULLS LAST, c.id DESC
		LIMIT 1`,
		userID, moduleID, model.ClaimApproved)
	return scanClaim(row)
}

func (r *claimRepository) NextWaitlistedForUpdate(ctx context.Context, db DBTX, batchID int) (*model.Claim, error) {
	row := db.QueryRow(ctx, `
		SELECT `+claimColumns+`
		FROM claims c
		WHERE c.batch_id = $1 AND c.status = $2
		ORDER BY c.created_at, c.id
		LIMIT 1
		FOR UPDATE SKIP LOCKED`,
		batchID, model.ClaimWaitlisted)
	return scanClaim(row)
}

func (r *claimRepository) UpdateStatus(ctx context.Context, db DBTX, id int, status string) (*model.Claim, error) {
	row := db.QueryRow(ctx, `
		UPDATE claims AS c
		SET status = $2::varchar,
		    decided_at = CASE WHEN $2::varchar IN ('approved', 'rejected') THEN now() ELSE c.decided_at END
		WHERE c.id = $1
		RETURNING `+claimColumns,
		id, status)
	return scanClaim(row)
}

func (r *claimRepository) ListByUser(ctx context.Context, db DBTX, filter model.ClaimListFilter) ([]model.ClaimDetail, error) {
	query := `
		SELECT ` + claimColumns + `, m.id, m.code, m.title, b.opens_at, b.closes_at
		FROM claims c
		JOIN batches b ON b.id = c.batch_id
		JOIN modules m ON m.id = b.module_id
		WHERE c.user_id = $1`
	args := []any{filter.UserID}
	query, args = appendClaimFilters(query, args, filter)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query riwayat klaim: %w", err)
	}
	defer rows.Close()

	result := make([]model.ClaimDetail, 0, filter.Limit+1)
	for rows.Next() {
		var d model.ClaimDetail
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.BatchID, &d.Status, &d.ClaimedAt, &d.DecidedAt, &d.CreatedAt,
			&d.ModuleID, &d.ModuleCode, &d.ModuleTitle, &d.BatchOpensAt, &d.BatchClosesAt,
		); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *claimRepository) ListByBatch(ctx context.Context, db DBTX, filter model.ClaimListFilter) ([]model.Claim, error) {
	query := `SELECT ` + claimColumns + ` FROM claims c WHERE c.batch_id = $1`
	args := []any{filter.BatchID}
	query, args = appendClaimFilters(query, args, filter)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query klaim per batch: %w", err)
	}
	defer rows.Close()

	result := make([]model.Claim, 0, filter.Limit+1)
	for rows.Next() {
		c, err := scanClaim(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *c)
	}
	return result, rows.Err()
}

func appendClaimFilters(query string, args []any, filter model.ClaimListFilter) (string, []any) {
	if filter.Status != "" {
		args = append(args, filter.Status)
		query += fmt.Sprintf(" AND c.status = $%d", len(args))
	}
	if filter.Cursor != nil {
		args = append(args, filter.Cursor.CreatedAt, filter.Cursor.ID)
		query += fmt.Sprintf(" AND (c.created_at, c.id) < ($%d, $%d)", len(args)-1, len(args))
	}
	args = append(args, filter.Limit+1)
	query += fmt.Sprintf(" ORDER BY c.created_at DESC, c.id DESC LIMIT $%d", len(args))
	return query, args
}

func (r *claimRepository) InsertLog(ctx context.Context, db DBTX, log model.ClaimLog) (*model.ClaimLog, error) {
	var out model.ClaimLog
	err := db.QueryRow(ctx, `
		INSERT INTO claim_logs (claim_id, actor_id, from_status, to_status, note)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, claim_id, actor_id, from_status, to_status, note, created_at`,
		log.ClaimID, log.ActorID, log.FromStatus, log.ToStatus, log.Note,
	).Scan(&out.ID, &out.ClaimID, &out.ActorID, &out.FromStatus, &out.ToStatus, &out.Note, &out.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("simpan log klaim: %w", err)
	}
	return &out, nil
}

func (r *claimRepository) ListLogs(ctx context.Context, db DBTX, claimID int) ([]model.ClaimLog, error) {
	rows, err := db.Query(ctx, `
		SELECT id, claim_id, actor_id, from_status, to_status, note, created_at
		FROM claim_logs
		WHERE claim_id = $1
		ORDER BY created_at, id`,
		claimID)
	if err != nil {
		return nil, fmt.Errorf("query log klaim: %w", err)
	}
	defer rows.Close()

	result := make([]model.ClaimLog, 0)
	for rows.Next() {
		var l model.ClaimLog
		if err := rows.Scan(&l.ID, &l.ClaimID, &l.ActorID, &l.FromStatus, &l.ToStatus, &l.Note, &l.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, rows.Err()
}
