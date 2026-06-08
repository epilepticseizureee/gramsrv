package domain

import "errors"

var (
	ErrPasswordHashInvalid   = errors.New("password hash invalid")
	ErrSRPIDInvalid          = errors.New("srp id invalid")
	ErrSRPPasswordChanged    = errors.New("srp password changed")
	ErrNewSettingsInvalid    = errors.New("new password settings invalid")
	ErrNewSaltInvalid        = errors.New("new password salt invalid")
	ErrPasswordRecoveryNA    = errors.New("password recovery not available")
	ErrEmailCodeInvalid      = errors.New("email code invalid")
	ErrEmailInvalid          = errors.New("email invalid")
	ErrSessionPasswordNeeded = errors.New("session password needed")
)

// PasswordKDFAlgo 是业务层的 SRP KDF 算法描述，不依赖 tg.*。
type PasswordKDFAlgo struct {
	Salt1 []byte
	Salt2 []byte
	G     int
	P     []byte
}

// SecurePasswordKDFAlgo 是 Telegram Passport secure secret 的 KDF 算法描述。
type SecurePasswordKDFAlgo struct {
	Kind string
	Salt []byte
}

// PasswordCheck 是 inputCheckPasswordEmpty/inputCheckPasswordSRP 的业务层表达。
type PasswordCheck struct {
	Empty bool
	SRPID int64
	A     []byte
	M1    []byte
}

// PasswordInputSettings 是 account.passwordInputSettings 的业务层表达。
type PasswordInputSettings struct {
	NewAlgo         *PasswordKDFAlgo
	NewPasswordHash []byte
	Hint            string
	HasHint         bool
	Email           string
	HasEmail        bool
}

// PrivatePasswordSettings 是 account.passwordSettings 的业务层表达。
type PrivatePasswordSettings struct {
	Email string
}

type PasswordResetKind string

const (
	PasswordResetOK            PasswordResetKind = "ok"
	PasswordResetRequestedWait PasswordResetKind = "requested_wait"
	PasswordResetFailedWait    PasswordResetKind = "failed_wait"
)

type PasswordResetResult struct {
	Kind      PasswordResetKind
	UntilDate int
	RetryDate int
}

// PasswordSettings 是账号 2FA/SRP 配置。默认 HasPassword=false。
type PasswordSettings struct {
	HasRecovery             bool
	HasSecureValues         bool
	HasPassword             bool
	CurrentAlgo             *PasswordKDFAlgo
	SRPB                    []byte
	SRPID                   int64
	Hint                    string
	EmailUnconfirmedPattern string
	RecoveryEmail           string
	LoginEmailPattern       string
	NewAlgo                 PasswordKDFAlgo
	NewSecureAlgo           SecurePasswordKDFAlgo
	SecureRandom            []byte
	PendingResetDate        int

	// Server-only SRP fields. They are persisted but never exposed to rpc/tg conversion.
	SRPVerifier []byte
	SRPBSecret  []byte

	RecoveryCode          string
	RecoveryCodeExpiresAt int64
}

// ReactionNotifyFrom stores one account-level reaction notification scope.
type ReactionNotifyFrom string

const (
	ReactionNotifyFromNone     ReactionNotifyFrom = "none"
	ReactionNotifyFromContacts ReactionNotifyFrom = "contacts"
	ReactionNotifyFromAll      ReactionNotifyFrom = "all"
)

// ReactionsNotifySettings stores the account reaction notification settings
// consumed by account.get/setReactionsNotifySettings.
type ReactionsNotifySettings struct {
	MessagesFrom  ReactionNotifyFrom
	StoriesFrom   ReactionNotifyFrom
	PollVotesFrom ReactionNotifyFrom
	ShowPreviews  bool
}

// PaidReactionPrivacyKind stores the account default paid reaction privacy.
type PaidReactionPrivacyKind string

const (
	PaidReactionPrivacyDefault   PaidReactionPrivacyKind = "default"
	PaidReactionPrivacyAnonymous PaidReactionPrivacyKind = "anonymous"
	PaidReactionPrivacyPeer      PaidReactionPrivacyKind = "peer"
)

// PaidReactionPrivacy is the domain representation of tg.PaidReactionPrivacy.
type PaidReactionPrivacy struct {
	Kind PaidReactionPrivacyKind
	Peer *Peer
}

// AccountReactionSettings groups account-level reaction preferences.
type AccountReactionSettings struct {
	Notify          ReactionsNotifySettings
	DefaultReaction MessageReaction
	PaidPrivacy     PaidReactionPrivacy
}

func DefaultAccountReactionSettings() AccountReactionSettings {
	return AccountReactionSettings{
		Notify: ReactionsNotifySettings{
			MessagesFrom:  ReactionNotifyFromContacts,
			StoriesFrom:   ReactionNotifyFromContacts,
			PollVotesFrom: ReactionNotifyFromContacts,
			ShowPreviews:  true,
		},
		DefaultReaction: MessageReaction{Type: MessageReactionEmoji, Emoticon: "👍"},
		PaidPrivacy:     PaidReactionPrivacy{Kind: PaidReactionPrivacyDefault},
	}
}
