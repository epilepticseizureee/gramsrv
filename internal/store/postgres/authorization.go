package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"telesrv/internal/domain"
	"telesrv/internal/store/postgres/sqlcgen"
)

// AuthorizationStore 用 PostgreSQL 实现 store.AuthorizationStore。
type AuthorizationStore struct {
	db sqlcgen.DBTX
	q  *sqlcgen.Queries
}

// NewAuthorizationStore 基于 pgx 连接池（或事务）创建 AuthorizationStore。
func NewAuthorizationStore(db sqlcgen.DBTX) *AuthorizationStore {
	return &AuthorizationStore{db: db, q: sqlcgen.New(db)}
}

func (s *AuthorizationStore) Bind(ctx context.Context, a domain.Authorization) error {
	if a.Hash == 0 {
		a.Hash = authorizationHash(a.AuthKeyID)
	}
	_, err := s.db.Exec(ctx, `
INSERT INTO authorizations (auth_key_id, user_id, hash, layer, device_model, platform, system_version, api_id, app_version, ip)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (auth_key_id) DO UPDATE SET
  user_id = EXCLUDED.user_id,
  hash = EXCLUDED.hash,
  layer = EXCLUDED.layer,
  device_model = EXCLUDED.device_model,
  platform = EXCLUDED.platform,
  system_version = EXCLUDED.system_version,
  api_id = EXCLUDED.api_id,
  app_version = EXCLUDED.app_version,
  ip = EXCLUDED.ip,
  active_at = now()`,
		authKeyIDToInt64(a.AuthKeyID), a.UserID, a.Hash, int32(a.Layer), a.DeviceModel, a.Platform, a.SystemVersion, int32(a.APIID), a.AppVersion, a.IP,
	)
	if err != nil {
		return fmt.Errorf("upsert authorization: %w", err)
	}
	return nil
}

func (s *AuthorizationStore) ByAuthKey(ctx context.Context, id [8]byte) (domain.Authorization, bool, error) {
	row, err := s.q.GetAuthorizationByAuthKey(ctx, authKeyIDToInt64(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Authorization{}, false, nil
		}
		return domain.Authorization{}, false, fmt.Errorf("get authorization: %w", err)
	}
	return domain.Authorization{
		AuthKeyID:     id,
		UserID:        row.UserID,
		Layer:         int(row.Layer),
		DeviceModel:   row.DeviceModel,
		Platform:      row.Platform,
		SystemVersion: row.SystemVersion,
		APIID:         int(row.ApiID),
		AppVersion:    row.AppVersion,
		IP:            row.Ip,
	}, true, nil
}

func (s *AuthorizationStore) ListByUser(ctx context.Context, userID int64) ([]domain.Authorization, error) {
	rows, err := s.q.ListAuthorizationsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list authorizations by user: %w", err)
	}
	out := make([]domain.Authorization, 0, len(rows))
	for _, row := range rows {
		out = append(out, authorizationFromRow(row))
	}
	return out, nil
}

func (s *AuthorizationStore) Delete(ctx context.Context, id [8]byte) error {
	if err := s.q.DeleteAuthorization(ctx, authKeyIDToInt64(id)); err != nil {
		return fmt.Errorf("delete authorization: %w", err)
	}
	return nil
}

func (s *AuthorizationStore) DeleteByHash(ctx context.Context, userID, hash int64) (domain.Authorization, bool, error) {
	row := s.db.QueryRow(ctx, `
DELETE FROM authorizations
WHERE user_id = $1 AND hash = $2
RETURNING auth_key_id, user_id, hash, layer, device_model, platform, system_version, api_id, app_version, ip, created_at, active_at`, userID, hash)
	var a domain.Authorization
	var authKeyID int64
	if err := row.Scan(
		&authKeyID, &a.UserID, &a.Hash, &a.Layer, &a.DeviceModel, &a.Platform, &a.SystemVersion,
		&a.APIID, &a.AppVersion, &a.IP, &a.CreatedAt, &a.ActiveAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Authorization{}, false, nil
		}
		return domain.Authorization{}, false, fmt.Errorf("delete authorization by hash: %w", err)
	}
	a.AuthKeyID = authKeyIDFromInt64(authKeyID)
	return a, true, nil
}

func (s *AuthorizationStore) DeleteByUserExcept(ctx context.Context, userID int64, keepAuthKeyID [8]byte) ([]domain.Authorization, error) {
	rows, err := s.db.Query(ctx, `
DELETE FROM authorizations
WHERE user_id = $1 AND auth_key_id <> $2
RETURNING auth_key_id, user_id, hash, layer, device_model, platform, system_version, api_id, app_version, ip, created_at, active_at`, userID, authKeyIDToInt64(keepAuthKeyID))
	if err != nil {
		return nil, fmt.Errorf("delete authorizations by user: %w", err)
	}
	defer rows.Close()
	out := make([]domain.Authorization, 0)
	for rows.Next() {
		var a domain.Authorization
		var authKeyID int64
		if err := rows.Scan(
			&authKeyID, &a.UserID, &a.Hash, &a.Layer, &a.DeviceModel, &a.Platform, &a.SystemVersion,
			&a.APIID, &a.AppVersion, &a.IP, &a.CreatedAt, &a.ActiveAt,
		); err != nil {
			return nil, fmt.Errorf("scan deleted authorization: %w", err)
		}
		a.AuthKeyID = authKeyIDFromInt64(authKeyID)
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deleted authorizations: %w", err)
	}
	return out, nil
}

func authorizationFromRow(row sqlcgen.Authorization) domain.Authorization {
	return domain.Authorization{
		AuthKeyID:     authKeyIDFromInt64(row.AuthKeyID),
		UserID:        row.UserID,
		Hash:          row.Hash,
		Layer:         int(row.Layer),
		DeviceModel:   row.DeviceModel,
		Platform:      row.Platform,
		SystemVersion: row.SystemVersion,
		APIID:         int(row.ApiID),
		AppVersion:    row.AppVersion,
		IP:            row.Ip,
		CreatedAt:     row.CreatedAt.Time,
		ActiveAt:      row.ActiveAt.Time,
	}
}

func authorizationHash(authKeyID [8]byte) int64 {
	sum := sha256.Sum256(authKeyID[:])
	hash := int64(binary.LittleEndian.Uint64(sum[:8]))
	if hash == 0 {
		return 1
	}
	return hash
}
