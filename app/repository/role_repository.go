package repository

import (
	"context"
	"fmt"
)

type RoleRepository interface {
	LoadPermissions(ctx context.Context, db DBTX) (map[string][]string, error)
}

type roleRepository struct{}

func NewRoleRepository() RoleRepository {
	return &roleRepository{}
}

func (r *roleRepository) LoadPermissions(ctx context.Context, db DBTX) (map[string][]string, error) {
	rows, err := db.Query(ctx, `
		SELECT r.name, COALESCE(rp.permission_name, '')
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_name = r.name
		ORDER BY r.name, rp.permission_name`)
	if err != nil {
		return nil, fmt.Errorf("mengambil permission: %w", err)
	}
	defer rows.Close()

	result := map[string][]string{}
	for rows.Next() {
		var role, permission string
		if err := rows.Scan(&role, &permission); err != nil {
			return nil, fmt.Errorf("membaca row permission: %w", err)
		}
		if _, ok := result[role]; !ok {
			result[role] = []string{}
		}
		if permission != "" {
			result[role] = append(result[role], permission)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca hasil query permission: %w", err)
	}
	return result, nil
}
