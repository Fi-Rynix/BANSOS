package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"bansos-backend/app/model"
)

var ErrModuleNotFound = errors.New("modul tidak ditemukan")

type ModuleRepository interface {
	Create(ctx context.Context, db DBTX, req model.CreateModuleRequest) (*model.Module, error)
	FindByID(ctx context.Context, db DBTX, id int) (*model.Module, error)
	FindByIDForUpdate(ctx context.Context, db DBTX, id int) (*model.Module, error)
	List(ctx context.Context, db DBTX, filter model.ModuleListFilter) ([]model.Module, error)
	Update(ctx context.Context, db DBTX, id int, req model.UpdateModuleRequest) (*model.Module, error)
	Deactivate(ctx context.Context, db DBTX, id int) error
	CountRunningBatches(ctx context.Context, db DBTX, moduleID int) (int, error)
}

type moduleRepository struct{}

func NewModuleRepository() ModuleRepository {
	return &moduleRepository{}
}

const moduleColumns = `m.id, m.code, m.title, m.category, m.level, m.description, m.is_active, m.created_at`

func scanModule(row pgx.Row) (*model.Module, error) {
	var m model.Module
	err := row.Scan(&m.ID, &m.Code, &m.Title, &m.Category, &m.Level, &m.Description, &m.IsActive, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrModuleNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *moduleRepository) Create(ctx context.Context, db DBTX, req model.CreateModuleRequest) (*model.Module, error) {
	row := db.QueryRow(ctx, `
		INSERT INTO modules AS m (code, title, category, level, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+moduleColumns,
		req.Code, req.Title, req.Category, req.Level, req.Description)
	return scanModule(row)
}

func (r *moduleRepository) FindByID(ctx context.Context, db DBTX, id int) (*model.Module, error) {
	row := db.QueryRow(ctx, `SELECT `+moduleColumns+` FROM modules m WHERE m.id = $1`, id)
	return scanModule(row)
}

func (r *moduleRepository) FindByIDForUpdate(ctx context.Context, db DBTX, id int) (*model.Module, error) {
	row := db.QueryRow(ctx, `SELECT `+moduleColumns+` FROM modules m WHERE m.id = $1 FOR UPDATE`, id)
	return scanModule(row)
}

func (r *moduleRepository) List(ctx context.Context, db DBTX, filter model.ModuleListFilter) ([]model.Module, error) {
	query := `SELECT ` + moduleColumns + ` FROM modules m WHERE TRUE`
	args := []any{}

	switch filter.Status {
	case model.ModuleStatusInactive:
		query += ` AND NOT m.is_active`
	case model.ModuleStatusAll:
	default:
		query += ` AND m.is_active`
	}
	if filter.Category != "" {
		args = append(args, filter.Category)
		query += fmt.Sprintf(" AND m.category = $%d", len(args))
	}
	if filter.Level != "" {
		args = append(args, filter.Level)
		query += fmt.Sprintf(" AND m.level = $%d", len(args))
	}
	if filter.Cursor != nil {
		args = append(args, filter.Cursor.CreatedAt, filter.Cursor.ID)
		query += fmt.Sprintf(" AND (m.created_at, m.id) < ($%d, $%d)", len(args)-1, len(args))
	}
	args = append(args, filter.Limit+1)
	query += fmt.Sprintf(" ORDER BY m.created_at DESC, m.id DESC LIMIT $%d", len(args))

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query katalog modul: %w", err)
	}
	defer rows.Close()

	result := make([]model.Module, 0, filter.Limit+1)
	for rows.Next() {
		m, err := scanModule(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *m)
	}
	return result, rows.Err()
}

func (r *moduleRepository) Update(ctx context.Context, db DBTX, id int, req model.UpdateModuleRequest) (*model.Module, error) {
	row := db.QueryRow(ctx, `
		UPDATE modules AS m
		SET code = $2, title = $3, category = $4, level = $5, description = $6, is_active = $7
		WHERE m.id = $1
		RETURNING `+moduleColumns,
		id, req.Code, req.Title, req.Category, req.Level, req.Description, *req.IsActive)
	return scanModule(row)
}

func (r *moduleRepository) Deactivate(ctx context.Context, db DBTX, id int) error {
	tag, err := db.Exec(ctx, `UPDATE modules SET is_active = FALSE WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menonaktifkan modul: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrModuleNotFound
	}
	return nil
}

// CountRunningBatches menghitung batch yang masih terjadwal atau sedang buka.
// Modul dengan batch seperti ini tidak boleh dinonaktifkan.
func (r *moduleRepository) CountRunningBatches(ctx context.Context, db DBTX, moduleID int) (int, error) {
	var n int
	err := db.QueryRow(ctx, `
		SELECT count(*) FROM batches
		WHERE module_id = $1 AND status IN ('scheduled', 'open')`,
		moduleID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("menghitung batch berjalan: %w", err)
	}
	return n, nil
}
