package store

import (
	"context"

	"telesrv/internal/domain"
)

// AuthorizationStore 持久化设备授权（auth_key ↔ user 绑定）。实现见 store/memory（测试替身）、store/postgres。
type AuthorizationStore interface {
	Bind(ctx context.Context, a domain.Authorization) error
	ByAuthKey(ctx context.Context, authKeyID [8]byte) (domain.Authorization, bool, error)
	ListByUser(ctx context.Context, userID int64) ([]domain.Authorization, error)
	Delete(ctx context.Context, authKeyID [8]byte) error
	DeleteByHash(ctx context.Context, userID, hash int64) (domain.Authorization, bool, error)
	DeleteByUserExcept(ctx context.Context, userID int64, keepAuthKeyID [8]byte) ([]domain.Authorization, error)
}
