package repository

import (
	"goplay/config"

	"context"
	"database/sql"
	"strings"
)

type sqlRepository struct {
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) Repository {
	return &sqlRepository{
		db: db,
	}
}

func OpenDatabase(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open(cfg.DB.DBName, cfg.DB.DBConn)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (r *sqlRepository) GetUsersById(ctx context.Context, ids []int) ([]PlayerInfo, error) {
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	query := `SELECT id, rating from table2 WHERE id IN (?` + strings.Repeat(",?", len(args)-1) + `)`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	playersInfo := make([]PlayerInfo, len(ids))
	rows.Scan(&playersInfo)

	return nil, nil
}
