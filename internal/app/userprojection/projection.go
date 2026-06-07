package userprojection

import (
	"context"

	"telesrv/internal/domain"
	"telesrv/internal/store"
)

// ProfilePhotoProvider returns current profile photos for a batch of owners.
type ProfilePhotoProvider interface {
	CurrentProfilePhotos(ctx context.Context, ownerType domain.PeerType, ownerIDs []int64) (map[int64]domain.ProfilePhotoRef, error)
}

// ProfilePhotoKindProvider returns current profile/fallback photos for a batch of owners.
type ProfilePhotoKindProvider interface {
	CurrentProfilePhotosKind(ctx context.Context, ownerType domain.PeerType, ownerIDs []int64, kind domain.ProfilePhotoKind) (map[int64]domain.ProfilePhotoRef, error)
}

// PrivacyEvaluator answers viewer-specific visibility for one user privacy key.
type PrivacyEvaluator interface {
	CanSee(ctx context.Context, ownerUserID, viewerUserID int64, key domain.PrivacyKey) (bool, error)
}

// Projector builds the current viewer's user view for RPC response payloads.
// It intentionally stays in app/domain types; tg.* conversion remains in rpc.
type Projector struct {
	contacts store.ContactStore
	photos   ProfilePhotoProvider
	privacy  PrivacyEvaluator
}

// Option configures a Projector.
type Option func(*Projector)

// WithContactStore enables viewer-specific contact name/phone projection.
func WithContactStore(c store.ContactStore) Option {
	return func(p *Projector) { p.contacts = c }
}

// WithPhotoProvider enables current profile photo enrichment.
func WithPhotoProvider(photos ProfilePhotoProvider) Option {
	return func(p *Projector) { p.photos = photos }
}

// WithPrivacyEvaluator enables profile/photo/status privacy projection.
func WithPrivacyEvaluator(privacy PrivacyEvaluator) Option {
	return func(p *Projector) { p.privacy = privacy }
}

// New creates a user projector.
func New(opts ...Option) *Projector {
	p := &Projector{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// ForViewer applies both current profile photos and owner-specific contact view.
func (p *Projector) ForViewer(ctx context.Context, viewerUserID int64, users []domain.User) ([]domain.User, error) {
	if p == nil {
		return users, nil
	}
	return projectBatch(ctx, p.contacts, p.photos, p.privacy, viewerUserID, users)
}

// One applies ForViewer to a single user.
func (p *Projector) One(ctx context.Context, viewerUserID int64, user domain.User) (domain.User, error) {
	if p == nil {
		return user, nil
	}
	projected, err := p.ForViewer(ctx, viewerUserID, []domain.User{user})
	if err != nil || len(projected) == 0 {
		return domain.User{}, err
	}
	return projected[0], nil
}

// WithProfilePhotos enriches users with their current avatar from profile photo storage.
// The lookup is best-effort: a storage error keeps the original user list.
func WithProfilePhotos(ctx context.Context, photos ProfilePhotoProvider, users []domain.User) []domain.User {
	if photos == nil || len(users) == 0 {
		return users
	}
	ids := make([]int64, 0, len(users))
	seen := make(map[int64]struct{}, len(users))
	for _, u := range users {
		if u.ID == 0 {
			continue
		}
		if _, ok := seen[u.ID]; ok {
			continue
		}
		seen[u.ID] = struct{}{}
		ids = append(ids, u.ID)
	}
	if len(ids) == 0 {
		return users
	}
	refs, err := photos.CurrentProfilePhotos(ctx, domain.PeerTypeUser, ids)
	if err != nil || len(refs) == 0 {
		return users
	}
	out := make([]domain.User, len(users))
	copy(out, users)
	for i := range out {
		if ref, ok := refs[out[i].ID]; ok {
			applyPhotoRef(&out[i], ref)
		}
	}
	return out
}

// ForViewer applies the owner-specific user view that Telegram clients expect.
// In particular, phone is visible for self and contacts; non-contacts should not
// receive a phone field because TDesktop will prefer it over the public name.
func ForViewer(ctx context.Context, contacts store.ContactStore, viewerUserID int64, users []domain.User) ([]domain.User, error) {
	if contacts == nil || viewerUserID == 0 || len(users) == 0 {
		return users, nil
	}
	out := make([]domain.User, len(users))
	copy(out, users)
	cache := make(map[int64]domain.User, len(users))
	for i := range out {
		u := out[i]
		if u.ID == 0 || u.ID == viewerUserID || u.ID == domain.OfficialSystemUserID {
			continue
		}
		if projected, ok := cache[u.ID]; ok {
			out[i] = projected
			continue
		}
		projected, err := projectOne(ctx, contacts, viewerUserID, u)
		if err != nil {
			return nil, err
		}
		cache[u.ID] = projected
		out[i] = projected
	}
	return out, nil
}

// One applies ForViewer to a single user.
func One(ctx context.Context, contacts store.ContactStore, viewerUserID int64, user domain.User) (domain.User, error) {
	projected, err := ForViewer(ctx, contacts, viewerUserID, []domain.User{user})
	if err != nil || len(projected) == 0 {
		return domain.User{}, err
	}
	return projected[0], nil
}

func projectBatch(ctx context.Context, contacts store.ContactStore, photos ProfilePhotoProvider, privacy PrivacyEvaluator, viewerUserID int64, users []domain.User) ([]domain.User, error) {
	if len(users) == 0 {
		return users, nil
	}
	out := make([]domain.User, len(users))
	copy(out, users)
	ids := uniqueUserIDs(out)
	profileRefs := map[int64]domain.ProfilePhotoRef{}
	fallbackRefs := map[int64]domain.ProfilePhotoRef{}
	personalRefs := map[int64]domain.ProfilePhotoRef{}
	if photos != nil && len(ids) > 0 {
		if kindPhotos, ok := photos.(ProfilePhotoKindProvider); ok {
			refs, err := kindPhotos.CurrentProfilePhotosKind(ctx, domain.PeerTypeUser, ids, domain.ProfilePhotoKindProfile)
			if err != nil {
				return nil, err
			}
			profileRefs = refs
			refs, err = kindPhotos.CurrentProfilePhotosKind(ctx, domain.PeerTypeUser, ids, domain.ProfilePhotoKindFallback)
			if err != nil {
				return nil, err
			}
			fallbackRefs = refs
		} else {
			refs, err := photos.CurrentProfilePhotos(ctx, domain.PeerTypeUser, ids)
			if err != nil {
				return nil, err
			}
			profileRefs = refs
		}
	}
	var contactsByID map[int64]domain.Contact
	if contacts != nil && viewerUserID != 0 && len(ids) > 0 {
		var err error
		contactsByID, err = contacts.GetMany(ctx, viewerUserID, ids)
		if err != nil {
			return nil, err
		}
		personalRefs, err = contacts.PersonalPhotos(ctx, viewerUserID, ids)
		if err != nil {
			return nil, err
		}
	}
	cache := make(map[int64]domain.User, len(out))
	for i := range out {
		u := out[i]
		if u.ID == 0 {
			continue
		}
		if projected, ok := cache[u.ID]; ok {
			out[i] = projected
			continue
		}
		projected := applyBasePhotos(u, profileRefs, fallbackRefs, personalRefs, viewerUserID)
		if viewerUserID != 0 && u.ID != viewerUserID && u.ID != domain.OfficialSystemUserID {
			contact, found := contactsByID[u.ID]
			projected = applyContactProjection(projected, contact, found)
			var err error
			projected, err = applyPrivacy(ctx, privacy, viewerUserID, projected, found, profileRefs, fallbackRefs, personalRefs)
			if err != nil {
				return nil, err
			}
		}
		cache[u.ID] = projected
		out[i] = projected
	}
	return out, nil
}

func projectOne(ctx context.Context, contacts store.ContactStore, viewerUserID int64, user domain.User) (domain.User, error) {
	contact, found, err := contacts.Get(ctx, viewerUserID, user.ID)
	if err != nil {
		return domain.User{}, err
	}
	if !found {
		user.Phone = ""
		user.Contact = false
		user.Mutual = false
		return user, nil
	}
	projected := user
	projected.Contact = true
	projected.Mutual = contact.Mutual || contact.User.Mutual
	if contact.User.Phone != "" {
		projected.Phone = contact.User.Phone
	} else {
		projected.Phone = contact.Phone
	}
	if contact.User.FirstName != "" || contact.User.LastName != "" {
		projected.FirstName = contact.User.FirstName
		projected.LastName = contact.User.LastName
	} else if contact.FirstName != "" || contact.LastName != "" {
		projected.FirstName = contact.FirstName
		projected.LastName = contact.LastName
	}
	return projected, nil
}

func uniqueUserIDs(users []domain.User) []int64 {
	seen := make(map[int64]struct{}, len(users))
	ids := make([]int64, 0, len(users))
	for _, user := range users {
		if user.ID == 0 {
			continue
		}
		if _, ok := seen[user.ID]; ok {
			continue
		}
		seen[user.ID] = struct{}{}
		ids = append(ids, user.ID)
	}
	return ids
}

func applyBasePhotos(user domain.User, profileRefs, fallbackRefs, personalRefs map[int64]domain.ProfilePhotoRef, viewerUserID int64) domain.User {
	if !hasPhotoLookups(profileRefs, fallbackRefs, personalRefs) {
		return user
	}
	clearPhoto(&user)
	if viewerUserID != 0 && user.ID != viewerUserID {
		if ref, ok := personalRefs[user.ID]; ok && ref.PhotoID != 0 {
			ref.Personal = true
			applyPhotoRef(&user, ref)
			return user
		}
	}
	if ref, ok := profileRefs[user.ID]; ok && ref.PhotoID != 0 {
		applyPhotoRef(&user, ref)
		return user
	}
	if ref, ok := fallbackRefs[user.ID]; ok && ref.PhotoID != 0 {
		applyPhotoRef(&user, ref)
	}
	return user
}

func applyContactProjection(user domain.User, contact domain.Contact, found bool) domain.User {
	if !found {
		user.Phone = ""
		user.Contact = false
		user.Mutual = false
		return user
	}
	user.Contact = true
	user.Mutual = contact.Mutual || contact.User.Mutual
	if contact.User.Phone != "" {
		user.Phone = contact.User.Phone
	} else {
		user.Phone = contact.Phone
	}
	if contact.User.FirstName != "" || contact.User.LastName != "" {
		user.FirstName = contact.User.FirstName
		user.LastName = contact.User.LastName
	} else if contact.FirstName != "" || contact.LastName != "" {
		user.FirstName = contact.FirstName
		user.LastName = contact.LastName
	}
	return user
}

func applyPrivacy(ctx context.Context, privacy PrivacyEvaluator, viewerUserID int64, user domain.User, isContact bool, profileRefs, fallbackRefs, personalRefs map[int64]domain.ProfilePhotoRef) (domain.User, error) {
	if privacy == nil {
		return user, nil
	}
	phoneAllowed, err := privacy.CanSee(ctx, user.ID, viewerUserID, domain.PrivacyKeyPhoneNumber)
	if err != nil {
		return domain.User{}, err
	}
	if !phoneAllowed && !isContact {
		user.Phone = ""
	}
	statusAllowed, err := privacy.CanSee(ctx, user.ID, viewerUserID, domain.PrivacyKeyStatusTimestamp)
	if err != nil {
		return domain.User{}, err
	}
	if !statusAllowed {
		user.LastSeenAt = 0
		if user.Status.Kind == domain.UserStatusOnline || user.Status.Kind == domain.UserStatusOffline {
			user.Status = domain.UserStatus{Kind: domain.UserStatusRecently}
		}
	}
	if ref, ok := personalRefs[user.ID]; ok && ref.PhotoID != 0 {
		ref.Personal = true
		applyPhotoRef(&user, ref)
		return user, nil
	}
	if !hasPhotoLookups(profileRefs, fallbackRefs, personalRefs) && user.PhotoID == 0 {
		return user, nil
	}
	profileAllowed, err := privacy.CanSee(ctx, user.ID, viewerUserID, domain.PrivacyKeyProfilePhoto)
	if err != nil {
		return domain.User{}, err
	}
	if profileAllowed {
		if ref, ok := profileRefs[user.ID]; ok && ref.PhotoID != 0 {
			applyPhotoRef(&user, ref)
			return user, nil
		}
	}
	if ref, ok := fallbackRefs[user.ID]; ok && ref.PhotoID != 0 {
		applyPhotoRef(&user, ref)
		return user, nil
	}
	clearPhoto(&user)
	return user, nil
}

func hasPhotoLookups(profileRefs, fallbackRefs, personalRefs map[int64]domain.ProfilePhotoRef) bool {
	return len(profileRefs) != 0 || len(fallbackRefs) != 0 || len(personalRefs) != 0
}

func applyPhotoRef(user *domain.User, ref domain.ProfilePhotoRef) {
	user.PhotoID = ref.PhotoID
	user.PhotoDCID = ref.DCID
	user.PhotoStripped = append([]byte(nil), ref.Stripped...)
	user.PhotoPersonal = ref.Personal
}

func clearPhoto(user *domain.User) {
	user.PhotoID = 0
	user.PhotoDCID = 0
	user.PhotoStripped = nil
	user.PhotoPersonal = false
}
