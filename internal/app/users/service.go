package users

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"telesrv/internal/app/userprojection"
	"telesrv/internal/domain"
	"telesrv/internal/store"
)

// ErrNotAuthorized 表示当前 auth_key 尚未登录。
var ErrNotAuthorized = errors.New("not authorized")

// ProfilePhotoProvider 批量返回用户当前头像（用于把 PhotoID/DCID/Stripped 富化到 domain.User）。
type ProfilePhotoProvider = userprojection.ProfilePhotoProvider

// Service 提供用户查询。
type Service struct {
	users     store.UserStore
	contacts  store.ContactStore
	photos    ProfilePhotoProvider
	privacy   userprojection.PrivacyEvaluator
	projector *userprojection.Projector
}

// Option 调整用户服务可选依赖。
type Option func(*Service)

// WithPhotoProvider 注入头像富化能力（缺省则用户不带头像）。
func WithPhotoProvider(p ProfilePhotoProvider) Option {
	return func(s *Service) { s.photos = p }
}

// WithContactStore enables viewer-specific contact name/phone projection.
func WithContactStore(c store.ContactStore) Option {
	return func(s *Service) { s.contacts = c }
}

// WithPrivacyEvaluator enables viewer-specific privacy projection.
func WithPrivacyEvaluator(p userprojection.PrivacyEvaluator) Option {
	return func(s *Service) { s.privacy = p }
}

const (
	minUsernameLen       = 5
	maxUsernameLen       = 32
	maxProfileNameRunes  = 64
	maxProfileAboutRunes = 70
	maxBatchUsers        = 1000
)

// NewService 创建用户服务。
func NewService(users store.UserStore, opts ...Option) *Service {
	s := &Service{users: users}
	for _, opt := range opts {
		opt(s)
	}
	s.projector = userprojection.New(
		userprojection.WithContactStore(s.contacts),
		userprojection.WithPhotoProvider(s.photos),
		userprojection.WithPrivacyEvaluator(s.privacy),
	)
	return s
}

// loadSelf 加载当前用户但不富化头像（供内部校验路径使用，避免无谓的头像查询）。
func (s *Service) loadSelf(ctx context.Context, userID int64) (domain.User, error) {
	if userID == 0 {
		return domain.User{}, ErrNotAuthorized
	}
	u, found, err := s.users.ByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	if !found {
		return domain.User{}, ErrNotAuthorized
	}
	return u, nil
}

// Self 返回当前登录的用户（带头像）。未登录返回 ErrNotAuthorized。
func (s *Service) Self(ctx context.Context, userID int64) (domain.User, error) {
	u, err := s.loadSelf(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	return s.projectOne(ctx, userID, u)
}

// ByID 返回指定用户。调用方必须已登录；access_hash 校验在 RPC 边界完成。
func (s *Service) ByID(ctx context.Context, currentUserID, userID int64) (domain.User, bool, error) {
	if currentUserID == 0 {
		return domain.User{}, false, ErrNotAuthorized
	}
	u, found, err := s.users.ByID(ctx, userID)
	if err != nil {
		return domain.User{}, false, err
	}
	if !found {
		return u, false, nil
	}
	u, err = s.projectOne(ctx, currentUserID, u)
	if err != nil {
		return domain.User{}, false, err
	}
	return u, true, nil
}

// ByIDs 批量返回指定用户。调用方必须已登录；缺失用户不会出现在结果中。
func (s *Service) ByIDs(ctx context.Context, currentUserID int64, userIDs []int64) ([]domain.User, error) {
	if currentUserID == 0 {
		return nil, ErrNotAuthorized
	}
	if len(userIDs) == 0 {
		return nil, nil
	}
	ids := make([]int64, 0, len(userIDs))
	seen := make(map[int64]struct{}, len(userIDs))
	for _, id := range userIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		if len(ids) >= maxBatchUsers {
			break
		}
	}
	users, err := s.users.ByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return s.projectUsers(ctx, currentUserID, users)
}

// CheckUsername 校验当前用户是否可以占用 username。
func (s *Service) CheckUsername(ctx context.Context, userID int64, username string) (bool, error) {
	self, err := s.loadSelf(ctx, userID)
	if err != nil {
		return false, err
	}
	username = normalizeUsername(username)
	if !validUsername(username) {
		return false, domain.ErrUsernameInvalid
	}
	u, found, err := s.users.ByUsername(ctx, username)
	if err != nil {
		return false, err
	}
	return !found || u.ID == self.ID, nil
}

// UpdateUsername 修改当前用户的主 username。空字符串表示删除 username。
func (s *Service) UpdateUsername(ctx context.Context, userID int64, username string) (domain.User, error) {
	self, err := s.loadSelf(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	username = normalizeUsername(username)
	if username != "" {
		if !validUsername(username) {
			return domain.User{}, domain.ErrUsernameInvalid
		}
		u, found, err := s.users.ByUsername(ctx, username)
		if err != nil {
			return domain.User{}, err
		}
		if found && u.ID != self.ID {
			return domain.User{}, domain.ErrUsernameOccupied
		}
	}
	if self.Username == username {
		return self, nil
	}
	u, err := s.users.UpdateUsername(ctx, self.ID, username)
	if err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// UpdateProfile 修改当前用户的基础资料。未设置的字段保持原值。
func (s *Service) UpdateProfile(ctx context.Context, userID int64, update domain.UserProfileUpdate) (domain.User, error) {
	self, err := s.loadSelf(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	firstName := self.FirstName
	lastName := self.LastName
	about := self.About
	if update.HasFirstName {
		firstName = strings.TrimSpace(update.FirstName)
	}
	if update.HasLastName {
		lastName = strings.TrimSpace(update.LastName)
	}
	if update.HasAbout {
		about = strings.TrimSpace(update.About)
	}
	if firstName == "" || utf8.RuneCountInString(firstName) > maxProfileNameRunes || utf8.RuneCountInString(lastName) > maxProfileNameRunes {
		return domain.User{}, domain.ErrFirstNameInvalid
	}
	if utf8.RuneCountInString(about) > maxProfileAboutRunes {
		return domain.User{}, domain.ErrAboutTooLong
	}
	if firstName == self.FirstName && lastName == self.LastName && about == self.About {
		return self, nil
	}
	return s.users.UpdateProfile(ctx, self.ID, firstName, lastName, about)
}

// UpdateLastSeen records the latest visible account activity time.
func (s *Service) UpdateLastSeen(ctx context.Context, userID int64, lastSeenAt int) error {
	if userID == 0 {
		return ErrNotAuthorized
	}
	if lastSeenAt <= 0 {
		return nil
	}
	return s.users.UpdateLastSeen(ctx, userID, lastSeenAt)
}

// ResolveUsername 解析 username 到用户；调用方必须已登录。
func (s *Service) ResolveUsername(ctx context.Context, currentUserID int64, username string) (domain.User, bool, error) {
	if _, err := s.loadSelf(ctx, currentUserID); err != nil {
		return domain.User{}, false, err
	}
	username = normalizeUsername(username)
	if !validUsername(username) {
		return domain.User{}, false, domain.ErrUsernameInvalid
	}
	u, found, err := s.users.ByUsername(ctx, username)
	if err != nil || !found {
		return u, found, err
	}
	u, err = s.projectOne(ctx, currentUserID, u)
	if err != nil {
		return domain.User{}, false, err
	}
	return u, true, nil
}

// ResolvePhone 解析手机号到用户；当前阶段默认允许手机号深链解析，隐私规则后续接 account privacy。
func (s *Service) ResolvePhone(ctx context.Context, currentUserID int64, phone string) (domain.User, bool, error) {
	if _, err := s.loadSelf(ctx, currentUserID); err != nil {
		return domain.User{}, false, err
	}
	phone = normalizePhone(phone)
	if phone == "" {
		return domain.User{}, false, domain.ErrPhoneNotOccupied
	}
	u, found, err := s.users.ByPhone(ctx, phone)
	if err != nil || !found {
		return u, found, err
	}
	u, err = s.projectOne(ctx, currentUserID, u)
	if err != nil {
		return domain.User{}, false, err
	}
	return u, true, nil
}

func (s *Service) projectUsers(ctx context.Context, viewerUserID int64, users []domain.User) ([]domain.User, error) {
	if s == nil || s.projector == nil {
		return users, nil
	}
	return s.projector.ForViewer(ctx, viewerUserID, users)
}

func (s *Service) projectOne(ctx context.Context, viewerUserID int64, user domain.User) (domain.User, error) {
	if s == nil || s.projector == nil {
		return user, nil
	}
	return s.projector.One(ctx, viewerUserID, user)
}

func normalizeUsername(username string) string {
	username = strings.TrimSpace(username)
	username = strings.TrimPrefix(username, "@")
	return strings.TrimSpace(username)
}

func validUsername(username string) bool {
	if len(username) < minUsernameLen || len(username) > maxUsernameLen {
		return false
	}
	for i := 0; i < len(username); i++ {
		c := username[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
			if i == 0 {
				return false
			}
		case c == '_':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(phone))
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
