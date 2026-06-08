package account

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"telesrv/internal/domain"
	"telesrv/internal/store"
)

var defaultSecureRandom = []byte("telesrv-tdesktop-dev-secure-rand")

const (
	passwordResetWait  = 7 * 24 * time.Hour
	passwordResetRetry = 24 * time.Hour
)

// EmailUnconfirmedError reports the dev recovery-code length expected by TDesktop.
type EmailUnconfirmedError struct {
	Length int
}

func (e EmailUnconfirmedError) Error() string {
	if e.Length <= 0 {
		return "email unconfirmed"
	}
	return fmt.Sprintf("email unconfirmed: %d", e.Length)
}

// Service 提供账号安全配置查询。
type Service struct {
	passwords store.PasswordStore
	reactions store.AccountReactionSettingsStore
}

// ServiceOption 调整 account 服务依赖。
type ServiceOption func(*Service)

// WithReactionSettings 注入账号级 reaction 设置持久化。
func WithReactionSettings(reactions store.AccountReactionSettingsStore) ServiceOption {
	return func(s *Service) {
		s.reactions = reactions
	}
}

// NewService 创建 account 服务。
func NewService(passwords store.PasswordStore, opts ...ServiceOption) *Service {
	s := &Service{passwords: passwords}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GetPassword 返回当前账号 2FA 配置。未登录或无记录时返回持久化策略的默认 no-password 配置。
func (s *Service) GetPassword(ctx context.Context, userID int64) (domain.PasswordSettings, error) {
	if s == nil || s.passwords == nil || userID == 0 {
		return defaultPasswordSettings(), nil
	}
	settings, found, err := s.passwords.GetByUser(ctx, userID)
	if err != nil {
		return domain.PasswordSettings{}, err
	}
	if !found {
		return defaultPasswordSettings(), nil
	}
	settings = normalizePasswordSettings(settings)
	if settings.HasPassword {
		secret, b, err := makeSRPChallenge(settings.SRPVerifier)
		if err != nil {
			return domain.PasswordSettings{}, err
		}
		settings.SRPBSecret = secret
		settings.SRPB = b
		if settings.SRPID == 0 {
			settings.SRPID, err = randomInt64()
			if err != nil {
				return domain.PasswordSettings{}, err
			}
		}
		if err := s.passwords.Save(ctx, userID, settings); err != nil {
			return domain.PasswordSettings{}, err
		}
	}
	return settings, nil
}

func defaultPasswordSettings() domain.PasswordSettings {
	return normalizePasswordSettings(domain.PasswordSettings{SecureRandom: append([]byte(nil), defaultSecureRandom...)})
}

func normalizePasswordSettings(settings domain.PasswordSettings) domain.PasswordSettings {
	if len(settings.SecureRandom) == 0 {
		settings.SecureRandom = append([]byte(nil), defaultSecureRandom...)
	}
	if len(settings.NewAlgo.P) == 0 {
		settings.NewAlgo = defaultPasswordAlgo()
	}
	if settings.NewSecureAlgo.Kind == "" {
		settings.NewSecureAlgo = defaultSecureAlgo()
	}
	if settings.HasPassword && settings.CurrentAlgo == nil {
		algo := settings.NewAlgo
		settings.CurrentAlgo = &algo
	}
	if settings.RecoveryEmail != "" {
		settings.HasRecovery = true
	}
	return settings
}

// CheckPassword validates the current account password check.
func (s *Service) CheckPassword(ctx context.Context, userID int64, check domain.PasswordCheck) error {
	settings, err := s.GetPasswordWithoutRefresh(ctx, userID)
	if err != nil {
		return err
	}
	return checkSRP(settings, check)
}

// GetPasswordWithoutRefresh returns persisted settings without rotating the SRP challenge.
func (s *Service) GetPasswordWithoutRefresh(ctx context.Context, userID int64) (domain.PasswordSettings, error) {
	if s == nil || s.passwords == nil || userID == 0 {
		return defaultPasswordSettings(), nil
	}
	settings, found, err := s.passwords.GetByUser(ctx, userID)
	if err != nil {
		return domain.PasswordSettings{}, err
	}
	if !found {
		return defaultPasswordSettings(), nil
	}
	return normalizePasswordSettings(settings), nil
}

// GetPasswordSettings validates the password and returns private 2FA settings.
func (s *Service) GetPasswordSettings(ctx context.Context, userID int64, check domain.PasswordCheck) (domain.PrivatePasswordSettings, error) {
	settings, err := s.GetPasswordWithoutRefresh(ctx, userID)
	if err != nil {
		return domain.PrivatePasswordSettings{}, err
	}
	if err := checkSRP(settings, check); err != nil {
		return domain.PrivatePasswordSettings{}, err
	}
	return domain.PrivatePasswordSettings{Email: settings.RecoveryEmail}, nil
}

// UpdatePasswordSettings sets, changes, clears, or updates the recovery email for 2FA.
func (s *Service) UpdatePasswordSettings(ctx context.Context, userID int64, check domain.PasswordCheck, input domain.PasswordInputSettings) error {
	if s == nil || s.passwords == nil || userID == 0 {
		return nil
	}
	settings, err := s.GetPasswordWithoutRefresh(ctx, userID)
	if err != nil {
		return err
	}
	if err := checkSRP(settings, check); err != nil {
		return err
	}
	if len(input.NewPasswordHash) == 0 && !input.HasEmail {
		settings = defaultPasswordSettings()
		settings.SecureRandom = randomBytesOrDefault(passwordHashSize, settings.SecureRandom)
		return s.passwords.Save(ctx, userID, settings)
	}
	if len(input.NewPasswordHash) > 0 {
		if err := validateNewPasswordSettings(input); err != nil {
			return err
		}
		srpID, err := randomInt64()
		if err != nil {
			return err
		}
		algo := *input.NewAlgo
		settings.CurrentAlgo = &algo
		settings.NewAlgo = defaultPasswordAlgo()
		settings.SRPVerifier = padToHash(input.NewPasswordHash)
		secret, b, err := makeSRPChallenge(settings.SRPVerifier)
		if err != nil {
			return err
		}
		settings.SRPBSecret = secret
		settings.SRPB = b
		settings.SRPID = srpID
		settings.HasPassword = true
		if input.HasHint {
			settings.Hint = input.Hint
		}
	}
	if input.HasEmail {
		email := strings.TrimSpace(input.Email)
		if email != "" && !strings.Contains(email, "@") {
			return domain.ErrEmailInvalid
		}
		settings.RecoveryEmail = email
		settings.HasRecovery = email != ""
		settings.LoginEmailPattern = emailPattern(email)
		settings.EmailUnconfirmedPattern = ""
	}
	settings.SecureRandom = randomBytesOrDefault(passwordHashSize, settings.SecureRandom)
	return s.passwords.Save(ctx, userID, normalizePasswordSettings(settings))
}

func (s *Service) RequestPasswordRecovery(ctx context.Context, userID int64) (string, error) {
	settings, err := s.GetPasswordWithoutRefresh(ctx, userID)
	if err != nil {
		return "", err
	}
	if !settings.HasPassword || settings.RecoveryEmail == "" {
		return "", domain.ErrPasswordRecoveryNA
	}
	settings.RecoveryCode = recoveryCode
	settings.RecoveryCodeExpiresAt = time.Now().Unix() + recoveryCodeTTL
	if s.passwords != nil {
		if err := s.passwords.Save(ctx, userID, settings); err != nil {
			return "", err
		}
	}
	return emailPattern(settings.RecoveryEmail), nil
}

func (s *Service) CheckRecoveryPassword(ctx context.Context, userID int64, code string) error {
	settings, err := s.GetPasswordWithoutRefresh(ctx, userID)
	if err != nil {
		return err
	}
	return checkRecoveryCode(settings, code)
}

func (s *Service) RecoverPassword(ctx context.Context, userID int64, code string, input *domain.PasswordInputSettings) error {
	if s == nil || s.passwords == nil || userID == 0 {
		return nil
	}
	settings, err := s.GetPasswordWithoutRefresh(ctx, userID)
	if err != nil {
		return err
	}
	if err := checkRecoveryCode(settings, code); err != nil {
		return err
	}
	if input == nil || len(input.NewPasswordHash) == 0 {
		settings = defaultPasswordSettings()
		return s.passwords.Save(ctx, userID, settings)
	}
	if err := validateNewPasswordSettings(*input); err != nil {
		return err
	}
	settings.CurrentAlgo = input.NewAlgo
	settings.SRPVerifier = padToHash(input.NewPasswordHash)
	settings.SRPID, err = randomInt64()
	if err != nil {
		return err
	}
	settings.SRPBSecret, settings.SRPB, err = makeSRPChallenge(settings.SRPVerifier)
	if err != nil {
		return err
	}
	settings.HasPassword = true
	if input.HasHint {
		settings.Hint = input.Hint
	}
	settings.RecoveryCode = ""
	settings.RecoveryCodeExpiresAt = 0
	return s.passwords.Save(ctx, userID, normalizePasswordSettings(settings))
}

func (s *Service) ResetPassword(ctx context.Context, userID int64) (domain.PasswordResetResult, error) {
	if s == nil || s.passwords == nil || userID == 0 {
		return domain.PasswordResetResult{Kind: domain.PasswordResetFailedWait, RetryDate: int(time.Now().Add(passwordResetRetry).Unix())}, nil
	}
	settings, err := s.GetPasswordWithoutRefresh(ctx, userID)
	if err != nil {
		return domain.PasswordResetResult{}, err
	}
	if !settings.HasPassword {
		return domain.PasswordResetResult{Kind: domain.PasswordResetOK}, nil
	}
	if settings.HasRecovery {
		return domain.PasswordResetResult{}, domain.ErrPasswordRecoveryNA
	}
	now := time.Now()
	if settings.PendingResetDate > 0 {
		if now.Unix() >= int64(settings.PendingResetDate) {
			next := defaultPasswordSettings()
			next.SecureRandom = randomBytesOrDefault(passwordHashSize, settings.SecureRandom)
			if err := s.passwords.Save(ctx, userID, next); err != nil {
				return domain.PasswordResetResult{}, err
			}
			return domain.PasswordResetResult{Kind: domain.PasswordResetOK}, nil
		}
		return domain.PasswordResetResult{Kind: domain.PasswordResetRequestedWait, UntilDate: settings.PendingResetDate}, nil
	}
	settings.PendingResetDate = int(now.Add(passwordResetWait).Unix())
	if err := s.passwords.Save(ctx, userID, normalizePasswordSettings(settings)); err != nil {
		return domain.PasswordResetResult{}, err
	}
	return domain.PasswordResetResult{Kind: domain.PasswordResetRequestedWait, UntilDate: settings.PendingResetDate}, nil
}

func (s *Service) DeclinePasswordReset(ctx context.Context, userID int64) error {
	if s == nil || s.passwords == nil || userID == 0 {
		return nil
	}
	settings, err := s.GetPasswordWithoutRefresh(ctx, userID)
	if err != nil {
		return err
	}
	settings.PendingResetDate = 0
	return s.passwords.Save(ctx, userID, normalizePasswordSettings(settings))
}

func (s *Service) ConfirmPasswordEmail(ctx context.Context, userID int64, code string) error {
	return s.CheckRecoveryPassword(ctx, userID, code)
}

func (s *Service) ResendPasswordEmail(ctx context.Context, userID int64) error {
	_, err := s.RequestPasswordRecovery(ctx, userID)
	return err
}

func (s *Service) CancelPasswordEmail(ctx context.Context, userID int64) error {
	settings, err := s.GetPasswordWithoutRefresh(ctx, userID)
	if err != nil {
		return err
	}
	settings.EmailUnconfirmedPattern = ""
	settings.RecoveryCode = ""
	settings.RecoveryCodeExpiresAt = 0
	if s.passwords != nil {
		return s.passwords.Save(ctx, userID, settings)
	}
	return nil
}

func checkRecoveryCode(settings domain.PasswordSettings, code string) error {
	if settings.RecoveryCode == "" {
		if code == recoveryCode {
			return nil
		}
		return domain.ErrPasswordRecoveryNA
	}
	if settings.RecoveryCodeExpiresAt > 0 && time.Now().Unix() > settings.RecoveryCodeExpiresAt {
		return domain.ErrEmailCodeInvalid
	}
	if subtle.ConstantTimeCompare([]byte(settings.RecoveryCode), []byte(code)) != 1 {
		return domain.ErrEmailCodeInvalid
	}
	return nil
}

func randomBytesOrDefault(n int, fallback []byte) []byte {
	out := make([]byte, n)
	if _, err := rand.Read(out); err != nil {
		return append([]byte(nil), fallback...)
	}
	return out
}

func randomInt64() (int64, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	out := int64(0)
	for _, v := range b {
		out = (out << 8) | int64(v)
	}
	if out == 0 {
		out = 1
	}
	if out < 0 {
		out = -out
	}
	return out, nil
}

func emailPattern(email string) string {
	if email == "" {
		return ""
	}
	at := strings.Index(email, "@")
	if at <= 1 {
		return email
	}
	name := email[:at]
	return name[:1] + "***" + name[len(name)-1:] + email[at:]
}

// GetReactionSettings returns account-level reaction preferences.
func (s *Service) GetReactionSettings(ctx context.Context, userID int64) (domain.AccountReactionSettings, error) {
	if s == nil || s.reactions == nil || userID == 0 {
		return domain.DefaultAccountReactionSettings(), nil
	}
	settings, found, err := s.reactions.GetReactionSettings(ctx, userID)
	if err != nil {
		return domain.AccountReactionSettings{}, err
	}
	if !found {
		return domain.DefaultAccountReactionSettings(), nil
	}
	return normalizeReactionSettings(settings), nil
}

// SetReactionsNotifySettings stores reaction notification preferences.
func (s *Service) SetReactionsNotifySettings(ctx context.Context, userID int64, notify domain.ReactionsNotifySettings) (domain.AccountReactionSettings, error) {
	settings, err := s.GetReactionSettings(ctx, userID)
	if err != nil {
		return domain.AccountReactionSettings{}, err
	}
	settings.Notify = normalizeNotifySettings(notify)
	return s.saveReactionSettings(ctx, userID, settings)
}

// SetDefaultReaction stores the account default quick reaction.
func (s *Service) SetDefaultReaction(ctx context.Context, userID int64, reaction domain.MessageReaction) (domain.AccountReactionSettings, error) {
	settings, err := s.GetReactionSettings(ctx, userID)
	if err != nil {
		return domain.AccountReactionSettings{}, err
	}
	if reaction.Type == "" || reaction.Emoticon == "" {
		reaction = domain.DefaultAccountReactionSettings().DefaultReaction
	}
	settings.DefaultReaction = reaction
	return s.saveReactionSettings(ctx, userID, settings)
}

// SetPaidReactionPrivacy stores the account default paid reaction privacy.
func (s *Service) SetPaidReactionPrivacy(ctx context.Context, userID int64, privacy domain.PaidReactionPrivacy) (domain.AccountReactionSettings, error) {
	settings, err := s.GetReactionSettings(ctx, userID)
	if err != nil {
		return domain.AccountReactionSettings{}, err
	}
	settings.PaidPrivacy = normalizePaidPrivacy(privacy)
	return s.saveReactionSettings(ctx, userID, settings)
}

func (s *Service) saveReactionSettings(ctx context.Context, userID int64, settings domain.AccountReactionSettings) (domain.AccountReactionSettings, error) {
	settings = normalizeReactionSettings(settings)
	if s == nil || s.reactions == nil || userID == 0 {
		return settings, nil
	}
	return settings, s.reactions.SaveReactionSettings(ctx, userID, settings)
}

func normalizeReactionSettings(settings domain.AccountReactionSettings) domain.AccountReactionSettings {
	defaults := domain.DefaultAccountReactionSettings()
	settings.Notify = normalizeNotifySettings(settings.Notify)
	if settings.DefaultReaction.Type == "" || settings.DefaultReaction.Emoticon == "" {
		settings.DefaultReaction = defaults.DefaultReaction
	}
	settings.PaidPrivacy = normalizePaidPrivacy(settings.PaidPrivacy)
	return settings
}

func normalizeNotifySettings(settings domain.ReactionsNotifySettings) domain.ReactionsNotifySettings {
	if !validNotifyFrom(settings.MessagesFrom) {
		settings.MessagesFrom = domain.ReactionNotifyFromContacts
	}
	if !validNotifyFrom(settings.StoriesFrom) {
		settings.StoriesFrom = domain.ReactionNotifyFromContacts
	}
	if !validNotifyFrom(settings.PollVotesFrom) {
		settings.PollVotesFrom = domain.ReactionNotifyFromContacts
	}
	return settings
}

func validNotifyFrom(value domain.ReactionNotifyFrom) bool {
	switch value {
	case domain.ReactionNotifyFromNone, domain.ReactionNotifyFromContacts, domain.ReactionNotifyFromAll:
		return true
	default:
		return false
	}
}

func normalizePaidPrivacy(privacy domain.PaidReactionPrivacy) domain.PaidReactionPrivacy {
	switch privacy.Kind {
	case domain.PaidReactionPrivacyAnonymous:
		return domain.PaidReactionPrivacy{Kind: domain.PaidReactionPrivacyAnonymous}
	case domain.PaidReactionPrivacyPeer:
		if privacy.Peer != nil && privacy.Peer.ID != 0 {
			peer := *privacy.Peer
			return domain.PaidReactionPrivacy{Kind: domain.PaidReactionPrivacyPeer, Peer: &peer}
		}
	}
	return domain.PaidReactionPrivacy{Kind: domain.PaidReactionPrivacyDefault}
}
