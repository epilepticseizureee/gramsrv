package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"telesrv/internal/domain"
	"telesrv/internal/store/postgres/sqlcgen"
)

// PasswordStore 用 PostgreSQL 实现 store.PasswordStore。
type PasswordStore struct {
	db sqlcgen.DBTX
	q  *sqlcgen.Queries
}

// NewPasswordStore 基于 pgx 连接池（或事务）创建 PasswordStore。
func NewPasswordStore(db sqlcgen.DBTX) *PasswordStore {
	return &PasswordStore{db: db, q: sqlcgen.New(db)}
}

func (s *PasswordStore) GetByUser(ctx context.Context, userID int64) (domain.PasswordSettings, bool, error) {
	row := s.db.QueryRow(ctx, `
SELECT
  has_recovery, has_secure_values, has_password, hint,
  email_unconfirmed_pattern, login_email_pattern, secure_random,
  current_algo_salt1, current_algo_salt2, current_algo_g, current_algo_p,
  srp_id, srp_verifier, srp_b_secret, srp_b,
  recovery_email, recovery_code, recovery_code_expires_at
FROM account_passwords
WHERE user_id = $1`, userID)
	var settings domain.PasswordSettings
	var salt1, salt2, p []byte
	var recoveryExpires sql.NullTime
	if err := row.Scan(
		&settings.HasRecovery, &settings.HasSecureValues, &settings.HasPassword, &settings.Hint,
		&settings.EmailUnconfirmedPattern, &settings.LoginEmailPattern, &settings.SecureRandom,
		&salt1, &salt2, &settings.NewAlgo.G, &p,
		&settings.SRPID, &settings.SRPVerifier, &settings.SRPBSecret, &settings.SRPB,
		&settings.RecoveryEmail, &settings.RecoveryCode, &recoveryExpires,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PasswordSettings{}, false, nil
		}
		return domain.PasswordSettings{}, false, fmt.Errorf("get account password: %w", err)
	}
	if len(salt1) > 0 || len(salt2) > 0 || len(p) > 0 || settings.NewAlgo.G != 0 {
		settings.CurrentAlgo = &domain.PasswordKDFAlgo{
			Salt1: append([]byte(nil), salt1...),
			Salt2: append([]byte(nil), salt2...),
			G:     settings.NewAlgo.G,
			P:     append([]byte(nil), p...),
		}
	}
	settings.NewAlgo.Salt1 = append([]byte(nil), salt1...)
	settings.NewAlgo.Salt2 = append([]byte(nil), salt2...)
	settings.NewAlgo.P = append([]byte(nil), p...)
	if recoveryExpires.Valid {
		settings.RecoveryCodeExpiresAt = recoveryExpires.Time.Unix()
	}
	settings.SecureRandom = append([]byte(nil), settings.SecureRandom...)
	settings.SRPVerifier = append([]byte(nil), settings.SRPVerifier...)
	settings.SRPBSecret = append([]byte(nil), settings.SRPBSecret...)
	settings.SRPB = append([]byte(nil), settings.SRPB...)
	return settings, true, nil
}

func (s *PasswordStore) Save(ctx context.Context, userID int64, settings domain.PasswordSettings) error {
	algo := settings.NewAlgo
	if settings.CurrentAlgo != nil {
		algo = *settings.CurrentAlgo
	}
	var recoveryExpires any
	if settings.RecoveryCodeExpiresAt > 0 {
		recoveryExpires = time.Unix(settings.RecoveryCodeExpiresAt, 0)
	}
	_, err := s.db.Exec(ctx, `
INSERT INTO account_passwords (
  user_id, has_recovery, has_secure_values, has_password, hint,
  email_unconfirmed_pattern, login_email_pattern, secure_random,
  current_algo_salt1, current_algo_salt2, current_algo_g, current_algo_p,
  srp_id, srp_verifier, srp_b_secret, srp_b,
  recovery_email, recovery_code, recovery_code_expires_at
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
ON CONFLICT (user_id) DO UPDATE SET
  has_recovery = EXCLUDED.has_recovery,
  has_secure_values = EXCLUDED.has_secure_values,
  has_password = EXCLUDED.has_password,
  hint = EXCLUDED.hint,
  email_unconfirmed_pattern = EXCLUDED.email_unconfirmed_pattern,
  login_email_pattern = EXCLUDED.login_email_pattern,
  secure_random = EXCLUDED.secure_random,
  current_algo_salt1 = EXCLUDED.current_algo_salt1,
  current_algo_salt2 = EXCLUDED.current_algo_salt2,
  current_algo_g = EXCLUDED.current_algo_g,
  current_algo_p = EXCLUDED.current_algo_p,
  srp_id = EXCLUDED.srp_id,
  srp_verifier = EXCLUDED.srp_verifier,
  srp_b_secret = EXCLUDED.srp_b_secret,
  srp_b = EXCLUDED.srp_b,
  recovery_email = EXCLUDED.recovery_email,
  recovery_code = EXCLUDED.recovery_code,
  recovery_code_expires_at = EXCLUDED.recovery_code_expires_at,
  updated_at = now()`,
		userID,
		settings.HasRecovery, settings.HasSecureValues, settings.HasPassword, settings.Hint,
		settings.EmailUnconfirmedPattern, settings.LoginEmailPattern, settings.SecureRandom,
		algo.Salt1, algo.Salt2, algo.G, algo.P,
		settings.SRPID, settings.SRPVerifier, settings.SRPBSecret, settings.SRPB,
		settings.RecoveryEmail, settings.RecoveryCode, recoveryExpires,
	)
	if err != nil {
		return fmt.Errorf("upsert account password: %w", err)
	}
	return nil
}

func (s *PasswordStore) GetReactionSettings(ctx context.Context, userID int64) (domain.AccountReactionSettings, bool, error) {
	row := s.db.QueryRow(ctx, `
SELECT messages_notify_from, stories_notify_from, poll_votes_notify_from, show_previews,
       default_reaction_type, default_reaction_value,
       paid_privacy_kind, paid_privacy_peer_type, paid_privacy_peer_id
FROM account_reaction_settings
WHERE user_id = $1`, userID)
	var messagesFrom, storiesFrom, pollVotesFrom string
	var defaultType, defaultValue string
	var paidKind string
	var paidPeerType sql.NullString
	var paidPeerID sql.NullInt64
	settings := domain.DefaultAccountReactionSettings()
	if err := row.Scan(
		&messagesFrom, &storiesFrom, &pollVotesFrom, &settings.Notify.ShowPreviews,
		&defaultType, &defaultValue, &paidKind, &paidPeerType, &paidPeerID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AccountReactionSettings{}, false, nil
		}
		return domain.AccountReactionSettings{}, false, fmt.Errorf("get account reaction settings: %w", err)
	}
	settings.Notify.MessagesFrom = domain.ReactionNotifyFrom(messagesFrom)
	settings.Notify.StoriesFrom = domain.ReactionNotifyFrom(storiesFrom)
	settings.Notify.PollVotesFrom = domain.ReactionNotifyFrom(pollVotesFrom)
	settings.DefaultReaction = domain.MessageReaction{Type: domain.MessageReactionType(defaultType), Emoticon: defaultValue}
	settings.PaidPrivacy = domain.PaidReactionPrivacy{Kind: domain.PaidReactionPrivacyKind(paidKind)}
	if settings.PaidPrivacy.Kind == domain.PaidReactionPrivacyPeer && paidPeerType.Valid && paidPeerID.Valid {
		peer := domain.Peer{Type: domain.PeerType(paidPeerType.String), ID: paidPeerID.Int64}
		settings.PaidPrivacy.Peer = &peer
	}
	return settings, true, nil
}

func (s *PasswordStore) SaveReactionSettings(ctx context.Context, userID int64, settings domain.AccountReactionSettings) error {
	var paidPeerType any
	var paidPeerID any
	if settings.PaidPrivacy.Kind == domain.PaidReactionPrivacyPeer && settings.PaidPrivacy.Peer != nil {
		paidPeerType = string(settings.PaidPrivacy.Peer.Type)
		paidPeerID = settings.PaidPrivacy.Peer.ID
	}
	if _, err := s.db.Exec(ctx, `
INSERT INTO account_reaction_settings (
    user_id, messages_notify_from, stories_notify_from, poll_votes_notify_from, show_previews,
    default_reaction_type, default_reaction_value, paid_privacy_kind, paid_privacy_peer_type, paid_privacy_peer_id
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (user_id) DO UPDATE SET
    messages_notify_from = EXCLUDED.messages_notify_from,
    stories_notify_from = EXCLUDED.stories_notify_from,
    poll_votes_notify_from = EXCLUDED.poll_votes_notify_from,
    show_previews = EXCLUDED.show_previews,
    default_reaction_type = EXCLUDED.default_reaction_type,
    default_reaction_value = EXCLUDED.default_reaction_value,
    paid_privacy_kind = EXCLUDED.paid_privacy_kind,
    paid_privacy_peer_type = EXCLUDED.paid_privacy_peer_type,
    paid_privacy_peer_id = EXCLUDED.paid_privacy_peer_id,
    updated_at = now()`,
		userID,
		string(settings.Notify.MessagesFrom), string(settings.Notify.StoriesFrom), string(settings.Notify.PollVotesFrom), settings.Notify.ShowPreviews,
		string(settings.DefaultReaction.Type), settings.DefaultReaction.Emoticon,
		string(settings.PaidPrivacy.Kind), paidPeerType, paidPeerID,
	); err != nil {
		return fmt.Errorf("save account reaction settings: %w", err)
	}
	return nil
}
