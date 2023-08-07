package repository

import (
	"context"
)

type PlayerInfo struct {
	ID     uint64
	Rating int
}

type Repository interface {
	GetUsersById(ctx context.Context, ids []int) ([]PlayerInfo, error)
}
