package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"bansos-backend/app/model"
)

var ErrBatchNotFound = errors.New("batch tidak ditemukan")

type BatchRepository interface {
	Create(ctx context.Context, db DBTX, moduleID int, req model.CreateBatchRequest) (*model.Batch, error)
	FindByID(ctx context.Context, db DBTX, id int) (*model.Batch, error)
	FindByIDForUpdate(ctx context.Context, db DBTX, id int) (*model.Batch, error)
	ListByModule(ctx context.Context, db DBTX, moduleID int, includeDraft bool) ([]model.Batch, error)
	Update(ctx context.Context, db DBTX, id int, req model.UpdateBatchRequest) (*model.Batch, error)
	UpdateStatus(ctx context.Context, db DBTX, id int, status string) (*model.Batch, error)
	Delete(ctx context.Context, db DBTX, id int) error
}

type batchRepository struct{}

func NewBatchRepository() BatchRepository {
	return &batchRepository{}
}

const batchColumns = `b.id, b.module_id, b.quota, b.allocated, b.opens_at, b.closes_at, b.status, b.created_at`

func scanBatch(row pgx.Row) (*model.Batch, error) {
	var b model.Batch
	err := row.Scan(&b.ID, &b.ModuleID, &b.Quota, &b.Allocated, &b.OpensAt, &b.ClosesAt, &b.Status, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBatchNotFound
	}
	if err != nil {
		return nil, err
	}
	b.Remaining = b.Quota - b.Allocated
	return &b, nil
}

func (r *batchRepository) Create(ctx context.Context, db DBTX, moduleID int, req model.CreateBatchRequest) (*model.Batch, error) {
	row := db.QueryRow(ctx, `
		INSERT INTO batches AS b (module_id, quota, opens_at, closes_at)
		VALUES ($1, $2, $3, $4)
		RETURNING `+batchColumns,
		moduleID, req.Quota, req.OpensAt, req.ClosesAt)
	return scanBatch(row)
}

func (r *batchRepository) FindByID(ctx context.Context, db DBTX, id int) (*model.Batch, error) {
	row := db.QueryRow(ctx, `SELECT `+batchColumns+` FROM batches b WHERE b.id = $1`, id)
	return scanBatch(row)
}

func (r *batchRepository) FindByIDForUpdate(ctx context.Context, db DBTX, id int) (*model.Batch, error) {
	row := db.QueryRow(ctx, `SELECT `+batchColumns+` FROM batches b WHERE b.id = $1 FOR UPDATE`, id)
	return scanBatch(row)
}

func (r *batchRepository) ListByModule(ctx context.Context, db DBTX, moduleID int, includeDraft bool) ([]model.Batch, error) {
	query := `SELECT ` + batchColumns + ` FROM batches b WHERE b.module_id = $1`
	if !includeDraft {
		query += ` AND b.status <> 'draft'`
	}
	query += ` ORDER BY b.opens_at DESC, b.id DESC`

	rows, err := db.Query(ctx, query, moduleID)
	if err != nil {
		return nil, fmt.Errorf("query batch per modul: %w", err)
	}
	defer rows.Close()

	result := []model.Batch{}
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *b)
	}
	return result, rows.Err()
}

func (r *batchRepository) Update(ctx context.Context, db DBTX, id int, req model.UpdateBatchRequest) (*model.Batch, error) {
	row := db.QueryRow(ctx, `
		UPDATE batches AS b
		SET quota = $2, opens_at = $3, closes_at = $4
		WHERE b.id = $1
		RETURNING `+batchColumns,
		id, req.Quota, req.OpensAt, req.ClosesAt)
	return scanBatch(row)
}

// UpdateStatus juga merapikan jendela waktu:
// - dibuka lebih awal: opens_at dimajukan ke sekarang supaya war langsung jalan.
// - open ditutup lebih awal: closes_at dimajukan ke sekarang.
func (r *batchRepository) UpdateStatus(ctx context.Context, db DBTX, id int, status string) (*model.Batch, error) {
	row := db.QueryRow(ctx, `
		UPDATE batches AS b
		SET status = $2::varchar,
		    opens_at = CASE WHEN $2::varchar = 'open' THEN LEAST(b.opens_at, now()) ELSE b.opens_at END,
		    closes_at = CASE WHEN $2::varchar = 'closed' AND b.status = 'open'
		                     THEN LEAST(b.closes_at, GREATEST(now(), b.opens_at + INTERVAL '1 second'))
		                     ELSE b.closes_at END
		WHERE b.id = $1
		RETURNING `+batchColumns,
		id, status)
	return scanBatch(row)
}

func (r *batchRepository) Delete(ctx context.Context, db DBTX, id int) error {
	tag, err := db.Exec(ctx, `DELETE FROM batches WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus batch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrBatchNotFound
	}
	return nil
}
