package privacy

import (
	"context"
	"slices"

	"telesrv/internal/domain"
	"telesrv/internal/store"
)

const maxPrivacyRules = 100

// Service owns account privacy rules and viewer-specific evaluation.
type Service struct {
	rules    store.PrivacyStore
	contacts store.ContactStore
}

func NewService(rules store.PrivacyStore, contacts store.ContactStore) *Service {
	return &Service{rules: rules, contacts: contacts}
}

func (s *Service) GetRules(ctx context.Context, ownerUserID int64, key domain.PrivacyKey) (domain.PrivacyRules, error) {
	if !ValidKey(key) {
		return domain.PrivacyRules{}, domain.ErrPrivacyKeyInvalid
	}
	if s == nil || s.rules == nil {
		return defaultRules(ownerUserID, key), nil
	}
	rules, ok, err := s.rules.GetPrivacyRules(ctx, ownerUserID, key)
	if err != nil {
		return domain.PrivacyRules{}, err
	}
	if !ok {
		return defaultRules(ownerUserID, key), nil
	}
	rules.OwnerUserID = ownerUserID
	rules.Key = key
	if len(rules.Rules) == 0 {
		rules.Rules = domain.DefaultPrivacyRules(key)
	}
	return cloneRules(rules), nil
}

func (s *Service) SetRules(ctx context.Context, ownerUserID int64, key domain.PrivacyKey, rules []domain.PrivacyRule) (domain.PrivacyRules, error) {
	if !ValidKey(key) {
		return domain.PrivacyRules{}, domain.ErrPrivacyKeyInvalid
	}
	if len(rules) == 0 {
		rules = domain.DefaultPrivacyRules(key)
	}
	if err := validateRules(rules); err != nil {
		return domain.PrivacyRules{}, err
	}
	out := domain.PrivacyRules{OwnerUserID: ownerUserID, Key: key, Rules: cloneRuleSlice(rules)}
	if s != nil && s.rules != nil {
		if err := s.rules.SetPrivacyRules(ctx, out); err != nil {
			return domain.PrivacyRules{}, err
		}
	}
	return out, nil
}

func (s *Service) AddAllowUser(ctx context.Context, ownerUserID int64, key domain.PrivacyKey, targetUserID int64) (domain.PrivacyRules, bool, error) {
	if targetUserID == 0 {
		return domain.PrivacyRules{}, false, domain.ErrPrivacyRuleInvalid
	}
	rules, err := s.GetRules(ctx, ownerUserID, key)
	if err != nil {
		return domain.PrivacyRules{}, false, err
	}
	for i := range rules.Rules {
		if rules.Rules[i].Kind != domain.PrivacyRuleAllowUsers {
			continue
		}
		if slices.Contains(rules.Rules[i].UserIDs, targetUserID) {
			return rules, false, nil
		}
		rules.Rules[i].UserIDs = append(rules.Rules[i].UserIDs, targetUserID)
		next, err := s.SetRules(ctx, ownerUserID, key, rules.Rules)
		return next, true, err
	}
	rules.Rules = append([]domain.PrivacyRule{{Kind: domain.PrivacyRuleAllowUsers, UserIDs: []int64{targetUserID}}}, rules.Rules...)
	next, err := s.SetRules(ctx, ownerUserID, key, rules.Rules)
	return next, true, err
}

func (s *Service) CanSee(ctx context.Context, ownerUserID, viewerUserID int64, key domain.PrivacyKey) (bool, error) {
	if ownerUserID == 0 || viewerUserID == 0 {
		return false, nil
	}
	if ownerUserID == viewerUserID {
		return true, nil
	}
	rules, err := s.GetRules(ctx, ownerUserID, key)
	if err != nil {
		return false, err
	}
	evalCtx := domain.PrivacyContext{
		OwnerUserID:  ownerUserID,
		ViewerUserID: viewerUserID,
	}
	if s != nil && s.contacts != nil {
		if _, found, err := s.contacts.Get(ctx, ownerUserID, viewerUserID); err != nil {
			return false, err
		} else if found {
			evalCtx.ViewerIsContact = true
		}
	}
	return Evaluate(rules, evalCtx), nil
}

func Evaluate(rules domain.PrivacyRules, ctx domain.PrivacyContext) bool {
	if ctx.OwnerUserID != 0 && ctx.OwnerUserID == ctx.ViewerUserID {
		return true
	}
	if len(rules.Rules) == 0 {
		rules.Rules = domain.DefaultPrivacyRules(rules.Key)
	}
	for _, rule := range rules.Rules {
		if explicitDisallowMatches(rule, ctx) {
			return false
		}
	}
	for _, rule := range rules.Rules {
		if explicitAllowMatches(rule, ctx) {
			return true
		}
	}
	for _, rule := range rules.Rules {
		switch rule.Kind {
		case domain.PrivacyRuleDisallowContacts:
			if ctx.ViewerIsContact {
				return false
			}
		case domain.PrivacyRuleAllowContacts:
			if ctx.ViewerIsContact {
				return true
			}
		}
	}
	for _, rule := range rules.Rules {
		switch rule.Kind {
		case domain.PrivacyRuleDisallowAll:
			return false
		case domain.PrivacyRuleAllowAll:
			return true
		}
	}
	return false
}

func ValidKey(key domain.PrivacyKey) bool {
	switch key {
	case domain.PrivacyKeyStatusTimestamp,
		domain.PrivacyKeyChatInvite,
		domain.PrivacyKeyPhoneCall,
		domain.PrivacyKeyPhoneP2P,
		domain.PrivacyKeyForwards,
		domain.PrivacyKeyProfilePhoto,
		domain.PrivacyKeyPhoneNumber,
		domain.PrivacyKeyAddedByPhone,
		domain.PrivacyKeyVoiceMessages,
		domain.PrivacyKeyAbout,
		domain.PrivacyKeyBirthday,
		domain.PrivacyKeyStarGiftsAutoSave,
		domain.PrivacyKeyNoPaidMessages,
		domain.PrivacyKeySavedMusic:
		return true
	default:
		return false
	}
}

func validateRules(rules []domain.PrivacyRule) error {
	if len(rules) > maxPrivacyRules {
		return domain.ErrPrivacyRuleInvalid
	}
	for _, rule := range rules {
		switch rule.Kind {
		case domain.PrivacyRuleAllowContacts,
			domain.PrivacyRuleAllowAll,
			domain.PrivacyRuleAllowUsers,
			domain.PrivacyRuleDisallowContacts,
			domain.PrivacyRuleDisallowAll,
			domain.PrivacyRuleDisallowUsers,
			domain.PrivacyRuleAllowChatParticipants,
			domain.PrivacyRuleDisallowChatParticipants,
			domain.PrivacyRuleAllowCloseFriends,
			domain.PrivacyRuleAllowPremium,
			domain.PrivacyRuleAllowBots,
			domain.PrivacyRuleDisallowBots:
		default:
			return domain.ErrPrivacyRuleInvalid
		}
	}
	return nil
}

func explicitDisallowMatches(rule domain.PrivacyRule, ctx domain.PrivacyContext) bool {
	switch rule.Kind {
	case domain.PrivacyRuleDisallowUsers:
		return slices.Contains(rule.UserIDs, ctx.ViewerUserID)
	case domain.PrivacyRuleDisallowChatParticipants:
		return intersects(rule.ChatIDs, ctx.SharedChatIDs)
	case domain.PrivacyRuleDisallowBots:
		return ctx.ViewerIsBot
	default:
		return false
	}
}

func explicitAllowMatches(rule domain.PrivacyRule, ctx domain.PrivacyContext) bool {
	switch rule.Kind {
	case domain.PrivacyRuleAllowUsers:
		return slices.Contains(rule.UserIDs, ctx.ViewerUserID)
	case domain.PrivacyRuleAllowChatParticipants:
		return intersects(rule.ChatIDs, ctx.SharedChatIDs)
	case domain.PrivacyRuleAllowCloseFriends:
		return ctx.ViewerCloseFriend
	case domain.PrivacyRuleAllowPremium:
		return ctx.ViewerIsPremium
	case domain.PrivacyRuleAllowBots:
		return ctx.ViewerIsBot
	default:
		return false
	}
}

func intersects(a, b []int64) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	set := make(map[int64]struct{}, len(a))
	for _, id := range a {
		set[id] = struct{}{}
	}
	for _, id := range b {
		if _, ok := set[id]; ok {
			return true
		}
	}
	return false
}

func defaultRules(ownerUserID int64, key domain.PrivacyKey) domain.PrivacyRules {
	return domain.PrivacyRules{
		OwnerUserID: ownerUserID,
		Key:         key,
		Rules:       domain.DefaultPrivacyRules(key),
	}
}

func cloneRules(in domain.PrivacyRules) domain.PrivacyRules {
	out := in
	out.Rules = cloneRuleSlice(in.Rules)
	return out
}

func cloneRuleSlice(in []domain.PrivacyRule) []domain.PrivacyRule {
	out := make([]domain.PrivacyRule, len(in))
	for i, rule := range in {
		out[i] = rule
		out[i].UserIDs = append([]int64(nil), rule.UserIDs...)
		out[i].ChatIDs = append([]int64(nil), rule.ChatIDs...)
	}
	return out
}
