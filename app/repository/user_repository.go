package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"bansos-backend/app/model"
)

var (
	ErrUserNotFound = errors.New("pengguna tidak ditemukan")
	ErrEmailTaken   = errors.New("email sudah dipakai")
)

const userColumns = `u.id, u.name, u.email, u.password, u.role, u.is_active, u.created_at`

type UserRepository interface {
	Create(ctx context.Context, db DBTX, user model.User) (*model.User, error)
	FindByID(ctx context.Context, db DBTX, id int) (*model.User, error)
	FindByEmail(ctx context.Context, db DBTX, email string) (*model.User, error)
	SetActive(ctx context.Context, db DBTX, id int, active bool) (*model.User, error)
	UpdateRole(ctx context.Context, db DBTX, id int, role string) (*model.User, error)
	List(ctx context.Context, db DBTX, filter model.UserListFilter) ([]model.User, error)
}

func scanUser(row pgx.Row) (*model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

type userRepository struct{}

func NewUserRepository() UserRepository {
	return &userRepository{}
}

func (r *userRepository) Create(ctx context.Context, db DBTX, user model.User) (*model.User, error) {
	row := db.QueryRow(ctx, `
		INSERT INTO users AS u (name, email, password, role, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+userColumns,
		user.Name, user.Email, user.Password, user.Role, user.IsActive)
	return scanUser(row)
}

func (r *userRepository) FindByID(ctx context.Context, db DBTX, id int) (*model.User, error) {
	row := db.QueryRow(ctx, `SELECT `+userColumns+` FROM users u WHERE u.id = $1`, id)
	return scanUser(row)
}

func (r *userRepository) FindByEmail(ctx context.Context, db DBTX, email string) (*model.User, error) {
	row := db.QueryRow(ctx, `SELECT `+userColumns+` FROM users u WHERE u.email = $1`, email)
	return scanUser(row)
}

func (r *userRepository) SetActive(ctx context.Context, db DBTX, id int, active bool) (*model.User, error) {
	row := db.QueryRow(ctx, `
		UPDATE users AS u SET is_active = $2, updated_at = now()
		WHERE u.id = $1
		RETURNING `+userColumns, id, active)
	return scanUser(row)
}

func (r *userRepository) UpdateRole(ctx context.Context, db DBTX, id int, role string) (*model.User, error) {
	row := db.QueryRow(ctx, `
		UPDATE users AS u SET role = $2, updated_at = now()
		WHERE u.id = $1
		RETURNING `+userColumns, id, role)
	return scanUser(row)
}

func (r *userRepository) List(ctx context.Context, db DBTX, filter model.UserListFilter) ([]model.User, error) {
	query := `SELECT ` + userColumns + ` FROM users u WHERE TRUE`
	args := []any{}

	if filter.Role != "" {
		args = append(args, filter.Role)
		query += fmt.Sprintf(" AND u.role = $%d", len(args))
	}
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		query += fmt.Sprintf(" AND (u.name ILIKE $%d OR u.email ILIKE $%[1]d)", len(args))
	}
	if filter.Cursor != nil {
		args = append(args, filter.Cursor.CreatedAt, filter.Cursor.ID)
		query += fmt.Sprintf(" AND (u.created_at, u.id) < ($%d, $%d)", len(args)-1, len(args))
	}
	args = append(args, filter.Limit+1)
	query += fmt.Sprintf(" ORDER BY u.created_at DESC, u.id DESC LIMIT $%d", len(args))

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query daftar pengguna: %w", err)
	}
	defer rows.Close()

	result := make([]model.User, 0, filter.Limit+1)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *u)
	}
	return result, rows.Err()
}
