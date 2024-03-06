package data

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Permissions []string

type PermissionsModel struct {
	DB *pgxpool.Pool
}

func (pm *PermissionsModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
		SELECT permissions.code
		FROM permissions 
		INNER JOIN users_permissions ON users_permissions.permissions_id = permissions.id
		INNER JOIN users ON users_permissions.user_id = users.id
		WHERE users.id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := pm.DB.Query(ctx, query, userID)

	if err != nil {
		return nil, err
	}

	permissions, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var permission string

		err := row.Scan(&permission)

		return permission, err
	})

	if err != nil {
		return nil, err
	}

	return permissions, nil
}

func (pm *PermissionsModel) AddForUser(userID int64, codes ...string) error {
	query := `
		INSERT INTO users_permissions
		SELECT $1, permissions.id 
		FROM permissions 
		WHERE permissions.code = ANY($2)`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := pm.DB.Exec(ctx, query, userID, codes)

	return err
}

func (p Permissions) Include(code string) bool {
	for _, c := range p {
		if c == code {
			return true
		}
	}

	return false
}
