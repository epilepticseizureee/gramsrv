package rpc

import (
	"context"
	"errors"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"

	"telesrv/internal/compat/tdesktop"
	"telesrv/internal/domain"
)

const legacyAccountRegisterDeviceTypeID uint32 = 0x637ea878

// registerAccount 注册 account.* RPC handler。
func (r *Router) registerAccount(d *tg.ServerDispatcher) {
	d.OnAccountRegisterDevice(func(ctx context.Context, req *tg.AccountRegisterDeviceRequest) (bool, error) {
		return true, nil
	})
	d.OnAccountUnregisterDevice(func(ctx context.Context, req *tg.AccountUnregisterDeviceRequest) (bool, error) {
		return true, nil
	})
	d.OnAccountCheckUsername(r.onAccountCheckUsername)
	d.OnAccountUpdateProfile(r.onAccountUpdateProfile)
	d.OnAccountUpdateUsername(r.onAccountUpdateUsername)
	d.OnAccountGetPassword(func(ctx context.Context) (*tg.AccountPassword, error) {
		if r.deps.Account == nil {
			return tgPassword(domain.PasswordSettings{SecureRandom: []byte("telesrv-tdesktop-dev-secure-rand")}), nil
		}
		userID, _, err := r.currentUserID(ctx)
		if err != nil {
			return nil, internalErr()
		}
		settings, err := r.deps.Account.GetPassword(ctx, userID)
		if err != nil {
			return nil, internalErr()
		}
		return tgPassword(settings), nil
	})
	d.OnAccountGetNotifySettings(func(ctx context.Context, peer tg.InputNotifyPeerClass) (*tg.PeerNotifySettings, error) {
		return tdesktop.NotifySettings(), nil
	})
	d.OnAccountUpdateNotifySettings(func(ctx context.Context, req *tg.AccountUpdateNotifySettingsRequest) (bool, error) {
		return true, nil
	})
	d.OnAccountResetNotifySettings(func(ctx context.Context) (bool, error) {
		return true, nil
	})
	d.OnAccountGetPrivacy(r.onAccountGetPrivacy)
	d.OnAccountSetPrivacy(r.onAccountSetPrivacy)
	d.OnAccountGetAuthorizations(r.onAccountGetAuthorizations)
	d.OnAccountResetAuthorization(r.onAccountResetAuthorization)
	d.OnAccountGetPasswordSettings(r.onAccountGetPasswordSettings)
	d.OnAccountUpdatePasswordSettings(r.onAccountUpdatePasswordSettings)
	d.OnAccountConfirmPasswordEmail(r.onAccountConfirmPasswordEmail)
	d.OnAccountResendPasswordEmail(r.onAccountResendPasswordEmail)
	d.OnAccountCancelPasswordEmail(r.onAccountCancelPasswordEmail)
	d.OnAccountGetDefaultEmojiStatuses(func(ctx context.Context, hash int64) (tg.AccountEmojiStatusesClass, error) {
		return tdesktop.DefaultEmojiStatuses(), nil
	})
	d.OnAccountGetCollectibleEmojiStatuses(func(ctx context.Context, hash int64) (tg.AccountEmojiStatusesClass, error) {
		return tdesktop.CollectibleEmojiStatuses(), nil
	})
	d.OnAccountGetDefaultGroupPhotoEmojis(func(ctx context.Context, hash int64) (tg.EmojiListClass, error) {
		return tdesktop.DefaultGroupPhotoEmojis(), nil
	})
	d.OnAccountGetConnectedBots(func(ctx context.Context) (*tg.AccountConnectedBots, error) {
		return tdesktop.ConnectedBots(), nil
	})
	d.OnAccountGetReactionsNotifySettings(r.onAccountGetReactionsNotifySettings)
	d.OnAccountSetReactionsNotifySettings(r.onAccountSetReactionsNotifySettings)
	d.OnAccountGetContactSignUpNotification(func(ctx context.Context) (bool, error) {
		return false, nil
	})
	d.OnAccountSetContactSignUpNotification(func(ctx context.Context, silent bool) (bool, error) {
		return true, nil
	})
	d.OnAccountGetThemes(func(ctx context.Context, req *tg.AccountGetThemesRequest) (tg.AccountThemesClass, error) {
		return tdesktop.AccountThemes(), nil
	})
	d.OnAccountGetRecentEmojiStatuses(func(ctx context.Context, hash int64) (tg.AccountEmojiStatusesClass, error) {
		return &tg.AccountEmojiStatuses{Hash: 0, Statuses: []tg.EmojiStatusClass{}}, nil
	})
	d.OnAccountClearRecentEmojiStatuses(func(ctx context.Context) (bool, error) {
		return true, nil
	})
	d.OnAccountUpdateEmojiStatus(func(ctx context.Context, emojistatus tg.EmojiStatusClass) (bool, error) {
		return true, nil
	})
	d.OnAccountGetDefaultProfilePhotoEmojis(func(ctx context.Context, hash int64) (tg.EmojiListClass, error) {
		return tdesktop.DefaultGroupPhotoEmojis(), nil
	})
	d.OnAccountGetDefaultBackgroundEmojis(func(ctx context.Context, hash int64) (tg.EmojiListClass, error) {
		return tdesktop.DefaultGroupPhotoEmojis(), nil
	})
	d.OnAccountGetChannelDefaultEmojiStatuses(func(ctx context.Context, hash int64) (tg.AccountEmojiStatusesClass, error) {
		return &tg.AccountEmojiStatuses{Hash: 0, Statuses: []tg.EmojiStatusClass{}}, nil
	})
	d.OnAccountGetChannelRestrictedStatusEmojis(func(ctx context.Context, hash int64) (tg.EmojiListClass, error) {
		return tdesktop.DefaultGroupPhotoEmojis(), nil
	})
	d.OnAccountSetContentSettings(func(ctx context.Context, req *tg.AccountSetContentSettingsRequest) (bool, error) {
		return true, nil
	})
	d.OnAccountGetContentSettings(func(ctx context.Context) (*tg.AccountContentSettings, error) {
		return tdesktop.ContentSettings(), nil
	})
	d.OnAccountGetGlobalPrivacySettings(func(ctx context.Context) (*tg.GlobalPrivacySettings, error) {
		return tdesktop.GlobalPrivacySettings(), nil
	})
	d.OnAccountSetGlobalPrivacySettings(func(ctx context.Context, settings tg.GlobalPrivacySettings) (*tg.GlobalPrivacySettings, error) {
		return &settings, nil
	})
	d.OnAccountGetPasskeys(func(ctx context.Context) (*tg.AccountPasskeys, error) {
		return tdesktop.Passkeys(), nil
	})
	d.OnAccountGetWebAuthorizations(func(ctx context.Context) (*tg.AccountWebAuthorizations, error) {
		return tdesktop.WebAuthorizations(), nil
	})
	d.OnAccountResetWebAuthorization(func(ctx context.Context, hash int64) (bool, error) {
		return true, nil
	})
	d.OnAccountResetWebAuthorizations(func(ctx context.Context) (bool, error) {
		return true, nil
	})
	d.OnAccountGetNotifyExceptions(func(ctx context.Context, req *tg.AccountGetNotifyExceptionsRequest) (tg.UpdatesClass, error) {
		return &tg.Updates{Updates: []tg.UpdateClass{}, Users: []tg.UserClass{}, Chats: []tg.ChatClass{}, Date: int(r.clock.Now().Unix())}, nil
	})
	d.OnAccountGetAutoDownloadSettings(func(ctx context.Context) (*tg.AccountAutoDownloadSettings, error) {
		return tdesktop.AutoDownloadSettings(), nil
	})
	d.OnAccountSaveAutoDownloadSettings(func(ctx context.Context, req *tg.AccountSaveAutoDownloadSettingsRequest) (bool, error) {
		return true, nil
	})
	d.OnAccountGetSavedMusicIDs(func(ctx context.Context, hash int64) (tg.AccountSavedMusicIDsClass, error) {
		if _, _, err := r.currentUserID(ctx); err != nil {
			return nil, internalErr()
		}
		return &tg.AccountSavedMusicIDs{IDs: []int64{}}, nil
	})
	d.OnAccountGetSavedRingtones(func(ctx context.Context, hash int64) (tg.AccountSavedRingtonesClass, error) {
		if _, _, err := r.currentUserID(ctx); err != nil {
			return nil, internalErr()
		}
		return &tg.AccountSavedRingtones{Hash: 0, Ringtones: []tg.DocumentClass{}}, nil
	})
	d.OnAccountGetAccountTTL(r.onAccountGetAccountTTL)
	d.OnAccountSetAccountTTL(func(ctx context.Context, ttl tg.AccountDaysTTL) (bool, error) {
		return true, nil
	})
	d.OnAccountSetAuthorizationTTL(func(ctx context.Context, authorizationttldays int) (bool, error) {
		return true, nil
	})
	d.OnAccountChangeAuthorizationSettings(func(ctx context.Context, req *tg.AccountChangeAuthorizationSettingsRequest) (bool, error) {
		return true, nil
	})
	d.OnAccountResetPassword(r.onAccountResetPassword)
	d.OnAccountDeclinePasswordReset(r.onAccountDeclinePasswordReset)
	d.OnAccountUpdateStatus(r.onAccountUpdateStatus)
}

func (r *Router) onAccountGetAuthorizations(ctx context.Context) (*tg.AccountAuthorizations, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return nil, internalErr()
	}
	if r.deps.Auth == nil {
		return tdesktop.Authorizations(), nil
	}
	authKeyID, _ := AuthKeyIDFrom(ctx)
	items, err := r.deps.Auth.ListAuthorizations(ctx, userID)
	if err != nil {
		return nil, internalErr()
	}
	out := &tg.AccountAuthorizations{Authorizations: make([]tg.Authorization, 0, len(items))}
	for _, item := range items {
		out.Authorizations = append(out.Authorizations, tgAuthorization(item, authKeyID, int(r.clock.Now().Unix())))
	}
	return out, nil
}

func (r *Router) onAccountResetAuthorization(ctx context.Context, hash int64) (bool, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return false, internalErr()
	}
	if r.deps.Auth == nil {
		return true, nil
	}
	deleted, found, err := r.deps.Auth.ResetAuthorization(ctx, userID, hash)
	if err != nil {
		return false, internalErr()
	}
	if !found {
		return true, nil
	}
	r.invalidateAuthUserCache(deleted.AuthKeyID)
	r.unbindAuthKey(deleted.AuthKeyID)
	_ = r.clearAuthKeyState(ctx, deleted.AuthKeyID)
	return true, nil
}

func (r *Router) onAccountGetPasswordSettings(ctx context.Context, password tg.InputCheckPasswordSRPClass) (*tg.AccountPasswordSettings, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return nil, internalErr()
	}
	settings, err := r.deps.Account.GetPasswordSettings(ctx, userID, domainPasswordCheck(password))
	if err != nil {
		return nil, passwordErr(err)
	}
	return tgPasswordSettings(settings), nil
}

func (r *Router) handleLegacyAccountRegisterDevice(ctx context.Context, b *bin.Buffer) (bin.Encoder, error) {
	if err := b.ConsumeID(legacyAccountRegisterDeviceTypeID); err != nil {
		return nil, err
	}
	if _, err := b.Int(); err != nil {
		return nil, err
	}
	if _, err := b.String(); err != nil {
		return nil, err
	}
	_, _, err := r.currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	return &tg.BoolTrue{}, nil
}

func (r *Router) onAccountUpdatePasswordSettings(ctx context.Context, req *tg.AccountUpdatePasswordSettingsRequest) (bool, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return false, internalErr()
	}
	input, err := domainPasswordInputSettings(req.NewSettings)
	if err != nil {
		return false, err
	}
	if err := r.deps.Account.UpdatePasswordSettings(ctx, userID, domainPasswordCheck(req.Password), input); err != nil {
		return false, passwordErr(err)
	}
	return true, nil
}

func (r *Router) onAccountConfirmPasswordEmail(ctx context.Context, code string) (bool, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return false, internalErr()
	}
	if err := r.deps.Account.ConfirmPasswordEmail(ctx, userID, code); err != nil {
		return false, passwordErr(err)
	}
	return true, nil
}

func (r *Router) onAccountResendPasswordEmail(ctx context.Context) (bool, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return false, internalErr()
	}
	if err := r.deps.Account.ResendPasswordEmail(ctx, userID); err != nil {
		return false, passwordErr(err)
	}
	return true, nil
}

func (r *Router) onAccountCancelPasswordEmail(ctx context.Context) (bool, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return false, internalErr()
	}
	if err := r.deps.Account.CancelPasswordEmail(ctx, userID); err != nil {
		return false, passwordErr(err)
	}
	return true, nil
}

func (r *Router) onAccountGetPrivacy(ctx context.Context, key tg.InputPrivacyKeyClass) (*tg.AccountPrivacyRules, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return nil, internalErr()
	}
	domainKey, ok := domainPrivacyKeyFromInput(key)
	if !ok {
		return nil, privacyKeyInvalidErr()
	}
	if r.deps.Privacy == nil {
		return tdesktop.PrivacyRules(key), nil
	}
	rules, err := r.deps.Privacy.GetRules(ctx, userID, domainKey)
	if err != nil {
		return nil, privacyErr(err)
	}
	return r.tgAccountPrivacyRules(ctx, userID, rules)
}

func (r *Router) onAccountSetPrivacy(ctx context.Context, req *tg.AccountSetPrivacyRequest) (*tg.AccountPrivacyRules, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return nil, internalErr()
	}
	domainKey, ok := domainPrivacyKeyFromInput(req.Key)
	if !ok {
		return nil, privacyKeyInvalidErr()
	}
	rules, err := r.domainPrivacyRulesFromInput(ctx, userID, req.Rules)
	if err != nil {
		return nil, err
	}
	if r.deps.Privacy == nil {
		return &tg.AccountPrivacyRules{Rules: tgPrivacyRules(rules), Users: []tg.UserClass{}, Chats: []tg.ChatClass{}}, nil
	}
	saved, err := r.deps.Privacy.SetRules(ctx, userID, domainKey, rules)
	if err != nil {
		return nil, privacyErr(err)
	}
	out, err := r.tgAccountPrivacyRules(ctx, userID, saved)
	if err != nil {
		return nil, err
	}
	r.pushUserUpdates(ctx, userID, &tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdatePrivacy{
			Key:   tgPrivacyKey(saved.Key),
			Rules: tgPrivacyRules(saved.Rules),
		}},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})
	return out, nil
}

func (r *Router) onAccountGetAccountTTL(ctx context.Context) (*tg.AccountDaysTTL, error) {
	if _, _, err := r.currentUserID(ctx); err != nil {
		return nil, internalErr()
	}
	return &tg.AccountDaysTTL{Days: 365}, nil
}

func (r *Router) onAccountResetPassword(ctx context.Context) (tg.AccountResetPasswordResultClass, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return nil, internalErr()
	}
	if r.deps.Account == nil {
		return &tg.AccountResetPasswordFailedWait{RetryDate: int(r.clock.Now().Unix()) + 86400}, nil
	}
	result, err := r.deps.Account.ResetPassword(ctx, userID)
	if err != nil {
		return nil, passwordErr(err)
	}
	return tgPasswordResetResult(result), nil
}

func (r *Router) onAccountDeclinePasswordReset(ctx context.Context) (bool, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return false, internalErr()
	}
	if r.deps.Account == nil {
		return true, nil
	}
	if err := r.deps.Account.DeclinePasswordReset(ctx, userID); err != nil {
		return false, internalErr()
	}
	return true, nil
}

func (r *Router) onAccountUpdateStatus(ctx context.Context, offline bool) (bool, error) {
	userID, authorized, err := r.currentUserID(ctx)
	if err != nil {
		return false, internalErr()
	}
	if !authorized || userID == 0 {
		return true, nil
	}
	status := r.setPresenceFromContext(ctx, userID, offline)
	r.pushUserStatus(ctx, userID, status)
	return true, nil
}

func tgPasswordResetResult(result domain.PasswordResetResult) tg.AccountResetPasswordResultClass {
	switch result.Kind {
	case domain.PasswordResetOK:
		return &tg.AccountResetPasswordOk{}
	case domain.PasswordResetRequestedWait:
		return &tg.AccountResetPasswordRequestedWait{UntilDate: result.UntilDate}
	case domain.PasswordResetFailedWait:
		return &tg.AccountResetPasswordFailedWait{RetryDate: result.RetryDate}
	default:
		return &tg.AccountResetPasswordFailedWait{}
	}
}

func (r *Router) tgAccountPrivacyRules(ctx context.Context, viewerUserID int64, rules domain.PrivacyRules) (*tg.AccountPrivacyRules, error) {
	userIDs := privacyRuleUserIDs(rules.Rules)
	users := []domain.User{}
	if r.deps.Users != nil && len(userIDs) > 0 {
		var err error
		users, err = r.deps.Users.ByIDs(ctx, viewerUserID, userIDs)
		if err != nil {
			return nil, internalErr()
		}
	}
	return &tg.AccountPrivacyRules{
		Rules: tgPrivacyRules(rules.Rules),
		Users: tgUsers(users),
		Chats: []tg.ChatClass{},
	}, nil
}

func (r *Router) domainPrivacyRulesFromInput(ctx context.Context, userID int64, in []tg.InputPrivacyRuleClass) ([]domain.PrivacyRule, error) {
	out := make([]domain.PrivacyRule, 0, len(in))
	for _, rule := range in {
		switch v := rule.(type) {
		case *tg.InputPrivacyValueAllowContacts:
			out = append(out, domain.PrivacyRule{Kind: domain.PrivacyRuleAllowContacts})
		case *tg.InputPrivacyValueAllowAll:
			out = append(out, domain.PrivacyRule{Kind: domain.PrivacyRuleAllowAll})
		case *tg.InputPrivacyValueAllowUsers:
			ids, err := r.privacyUserIDsFromInput(ctx, userID, v.Users)
			if err != nil {
				return nil, err
			}
			out = append(out, domain.PrivacyRule{Kind: domain.PrivacyRuleAllowUsers, UserIDs: ids})
		case *tg.InputPrivacyValueDisallowContacts:
			out = append(out, domain.PrivacyRule{Kind: domain.PrivacyRuleDisallowContacts})
		case *tg.InputPrivacyValueDisallowAll:
			out = append(out, domain.PrivacyRule{Kind: domain.PrivacyRuleDisallowAll})
		case *tg.InputPrivacyValueDisallowUsers:
			ids, err := r.privacyUserIDsFromInput(ctx, userID, v.Users)
			if err != nil {
				return nil, err
			}
			out = append(out, domain.PrivacyRule{Kind: domain.PrivacyRuleDisallowUsers, UserIDs: ids})
		case *tg.InputPrivacyValueAllowChatParticipants:
			out = append(out, domain.PrivacyRule{Kind: domain.PrivacyRuleAllowChatParticipants, ChatIDs: append([]int64(nil), v.Chats...)})
		case *tg.InputPrivacyValueDisallowChatParticipants:
			out = append(out, domain.PrivacyRule{Kind: domain.PrivacyRuleDisallowChatParticipants, ChatIDs: append([]int64(nil), v.Chats...)})
		case *tg.InputPrivacyValueAllowCloseFriends:
			out = append(out, domain.PrivacyRule{Kind: domain.PrivacyRuleAllowCloseFriends})
		case *tg.InputPrivacyValueAllowPremium:
			out = append(out, domain.PrivacyRule{Kind: domain.PrivacyRuleAllowPremium})
		case *tg.InputPrivacyValueAllowBots:
			out = append(out, domain.PrivacyRule{Kind: domain.PrivacyRuleAllowBots})
		case *tg.InputPrivacyValueDisallowBots:
			out = append(out, domain.PrivacyRule{Kind: domain.PrivacyRuleDisallowBots})
		default:
			return nil, privacyValueInvalidErr()
		}
	}
	return out, nil
}

func (r *Router) privacyUserIDsFromInput(ctx context.Context, currentUserID int64, inputs []tg.InputUserClass) ([]int64, error) {
	out := make([]int64, 0, len(inputs))
	seen := make(map[int64]struct{}, len(inputs))
	for _, input := range inputs {
		u, found, err := r.userFromInput(ctx, currentUserID, input)
		if err != nil {
			return nil, internalErr()
		}
		if !found || u.ID == 0 {
			return nil, userIDInvalidErr()
		}
		if _, ok := seen[u.ID]; ok {
			continue
		}
		seen[u.ID] = struct{}{}
		out = append(out, u.ID)
	}
	return out, nil
}

func domainPrivacyKeyFromInput(key tg.InputPrivacyKeyClass) (domain.PrivacyKey, bool) {
	switch key.(type) {
	case *tg.InputPrivacyKeyStatusTimestamp:
		return domain.PrivacyKeyStatusTimestamp, true
	case *tg.InputPrivacyKeyChatInvite:
		return domain.PrivacyKeyChatInvite, true
	case *tg.InputPrivacyKeyPhoneCall:
		return domain.PrivacyKeyPhoneCall, true
	case *tg.InputPrivacyKeyPhoneP2P:
		return domain.PrivacyKeyPhoneP2P, true
	case *tg.InputPrivacyKeyForwards:
		return domain.PrivacyKeyForwards, true
	case *tg.InputPrivacyKeyProfilePhoto:
		return domain.PrivacyKeyProfilePhoto, true
	case *tg.InputPrivacyKeyPhoneNumber:
		return domain.PrivacyKeyPhoneNumber, true
	case *tg.InputPrivacyKeyAddedByPhone:
		return domain.PrivacyKeyAddedByPhone, true
	case *tg.InputPrivacyKeyVoiceMessages:
		return domain.PrivacyKeyVoiceMessages, true
	case *tg.InputPrivacyKeyAbout:
		return domain.PrivacyKeyAbout, true
	case *tg.InputPrivacyKeyBirthday:
		return domain.PrivacyKeyBirthday, true
	case *tg.InputPrivacyKeyStarGiftsAutoSave:
		return domain.PrivacyKeyStarGiftsAutoSave, true
	case *tg.InputPrivacyKeyNoPaidMessages:
		return domain.PrivacyKeyNoPaidMessages, true
	case *tg.InputPrivacyKeySavedMusic:
		return domain.PrivacyKeySavedMusic, true
	default:
		return "", false
	}
}

func tgPrivacyKey(key domain.PrivacyKey) tg.PrivacyKeyClass {
	switch key {
	case domain.PrivacyKeyStatusTimestamp:
		return &tg.PrivacyKeyStatusTimestamp{}
	case domain.PrivacyKeyChatInvite:
		return &tg.PrivacyKeyChatInvite{}
	case domain.PrivacyKeyPhoneCall:
		return &tg.PrivacyKeyPhoneCall{}
	case domain.PrivacyKeyPhoneP2P:
		return &tg.PrivacyKeyPhoneP2P{}
	case domain.PrivacyKeyForwards:
		return &tg.PrivacyKeyForwards{}
	case domain.PrivacyKeyProfilePhoto:
		return &tg.PrivacyKeyProfilePhoto{}
	case domain.PrivacyKeyPhoneNumber:
		return &tg.PrivacyKeyPhoneNumber{}
	case domain.PrivacyKeyAddedByPhone:
		return &tg.PrivacyKeyAddedByPhone{}
	case domain.PrivacyKeyVoiceMessages:
		return &tg.PrivacyKeyVoiceMessages{}
	case domain.PrivacyKeyAbout:
		return &tg.PrivacyKeyAbout{}
	case domain.PrivacyKeyBirthday:
		return &tg.PrivacyKeyBirthday{}
	case domain.PrivacyKeyStarGiftsAutoSave:
		return &tg.PrivacyKeyStarGiftsAutoSave{}
	case domain.PrivacyKeyNoPaidMessages:
		return &tg.PrivacyKeyNoPaidMessages{}
	case domain.PrivacyKeySavedMusic:
		return &tg.PrivacyKeySavedMusic{}
	default:
		return &tg.PrivacyKeyStatusTimestamp{}
	}
}

func tgPrivacyRules(rules []domain.PrivacyRule) []tg.PrivacyRuleClass {
	out := make([]tg.PrivacyRuleClass, 0, len(rules))
	for _, rule := range rules {
		switch rule.Kind {
		case domain.PrivacyRuleAllowContacts:
			out = append(out, &tg.PrivacyValueAllowContacts{})
		case domain.PrivacyRuleAllowAll:
			out = append(out, &tg.PrivacyValueAllowAll{})
		case domain.PrivacyRuleAllowUsers:
			out = append(out, &tg.PrivacyValueAllowUsers{Users: append([]int64(nil), rule.UserIDs...)})
		case domain.PrivacyRuleDisallowContacts:
			out = append(out, &tg.PrivacyValueDisallowContacts{})
		case domain.PrivacyRuleDisallowAll:
			out = append(out, &tg.PrivacyValueDisallowAll{})
		case domain.PrivacyRuleDisallowUsers:
			out = append(out, &tg.PrivacyValueDisallowUsers{Users: append([]int64(nil), rule.UserIDs...)})
		case domain.PrivacyRuleAllowChatParticipants:
			out = append(out, &tg.PrivacyValueAllowChatParticipants{Chats: append([]int64(nil), rule.ChatIDs...)})
		case domain.PrivacyRuleDisallowChatParticipants:
			out = append(out, &tg.PrivacyValueDisallowChatParticipants{Chats: append([]int64(nil), rule.ChatIDs...)})
		case domain.PrivacyRuleAllowCloseFriends:
			out = append(out, &tg.PrivacyValueAllowCloseFriends{})
		case domain.PrivacyRuleAllowPremium:
			out = append(out, &tg.PrivacyValueAllowPremium{})
		case domain.PrivacyRuleAllowBots:
			out = append(out, &tg.PrivacyValueAllowBots{})
		case domain.PrivacyRuleDisallowBots:
			out = append(out, &tg.PrivacyValueDisallowBots{})
		}
	}
	return out
}

func privacyRuleUserIDs(rules []domain.PrivacyRule) []int64 {
	seen := map[int64]struct{}{}
	out := make([]int64, 0)
	for _, rule := range rules {
		for _, id := range rule.UserIDs {
			if id == 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

func privacyErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrPrivacyKeyInvalid):
		return privacyKeyInvalidErr()
	case errors.Is(err, domain.ErrPrivacyRuleInvalid):
		return privacyValueInvalidErr()
	default:
		return internalErr()
	}
}

type accountReactionSettingsService interface {
	GetReactionSettings(ctx context.Context, userID int64) (domain.AccountReactionSettings, error)
	SetReactionsNotifySettings(ctx context.Context, userID int64, settings domain.ReactionsNotifySettings) (domain.AccountReactionSettings, error)
}

func (r *Router) onAccountGetReactionsNotifySettings(ctx context.Context) (*tg.ReactionsNotifySettings, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return nil, internalErr()
	}
	if svc, ok := r.deps.Account.(accountReactionSettingsService); ok {
		settings, err := svc.GetReactionSettings(ctx, userID)
		if err != nil {
			return nil, internalErr()
		}
		return tgReactionsNotifySettings(settings.Notify), nil
	}
	return tgReactionsNotifySettings(domain.DefaultAccountReactionSettings().Notify), nil
}

func (r *Router) onAccountSetReactionsNotifySettings(ctx context.Context, settings tg.ReactionsNotifySettings) (*tg.ReactionsNotifySettings, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return nil, internalErr()
	}
	notify := domainReactionsNotifySettings(settings)
	if svc, ok := r.deps.Account.(accountReactionSettingsService); ok {
		next, err := svc.SetReactionsNotifySettings(ctx, userID, notify)
		if err != nil {
			return nil, internalErr()
		}
		return tgReactionsNotifySettings(next.Notify), nil
	}
	return tgReactionsNotifySettings(notify), nil
}

func domainReactionsNotifySettings(settings tg.ReactionsNotifySettings) domain.ReactionsNotifySettings {
	return domain.ReactionsNotifySettings{
		MessagesFrom:  domainReactionNotifyFrom(settings.GetMessagesNotifyFrom),
		StoriesFrom:   domainReactionNotifyFrom(settings.GetStoriesNotifyFrom),
		PollVotesFrom: domainReactionNotifyFrom(settings.GetPollVotesNotifyFrom),
		ShowPreviews:  settings.ShowPreviews,
	}
}

func domainReactionNotifyFrom(get func() (tg.ReactionNotificationsFromClass, bool)) domain.ReactionNotifyFrom {
	if get == nil {
		return domain.ReactionNotifyFromNone
	}
	value, ok := get()
	if !ok || value == nil {
		return domain.ReactionNotifyFromNone
	}
	switch value.(type) {
	case *tg.ReactionNotificationsFromAll:
		return domain.ReactionNotifyFromAll
	case *tg.ReactionNotificationsFromContacts:
		return domain.ReactionNotifyFromContacts
	default:
		return domain.ReactionNotifyFromNone
	}
}

func tgReactionsNotifySettings(settings domain.ReactionsNotifySettings) *tg.ReactionsNotifySettings {
	out := &tg.ReactionsNotifySettings{
		Sound:        &tg.NotificationSoundDefault{},
		ShowPreviews: settings.ShowPreviews,
	}
	if value := tgReactionNotifyFrom(settings.MessagesFrom); value != nil {
		out.SetMessagesNotifyFrom(value)
	}
	if value := tgReactionNotifyFrom(settings.StoriesFrom); value != nil {
		out.SetStoriesNotifyFrom(value)
	}
	if value := tgReactionNotifyFrom(settings.PollVotesFrom); value != nil {
		out.SetPollVotesNotifyFrom(value)
	}
	return out
}

func tgReactionNotifyFrom(value domain.ReactionNotifyFrom) tg.ReactionNotificationsFromClass {
	switch value {
	case domain.ReactionNotifyFromAll:
		return &tg.ReactionNotificationsFromAll{}
	case domain.ReactionNotifyFromContacts:
		return &tg.ReactionNotificationsFromContacts{}
	default:
		return nil
	}
}

func (r *Router) onAccountUpdateProfile(ctx context.Context, req *tg.AccountUpdateProfileRequest) (tg.UserClass, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return nil, internalErr()
	}
	svc, ok := r.deps.Users.(UserIdentityService)
	if !ok {
		return nil, internalErr()
	}
	firstName, hasFirstName := req.GetFirstName()
	lastName, hasLastName := req.GetLastName()
	about, hasAbout := req.GetAbout()
	u, err := svc.UpdateProfile(ctx, userID, domain.UserProfileUpdate{
		FirstName:    firstName,
		HasFirstName: hasFirstName,
		LastName:     lastName,
		HasLastName:  hasLastName,
		About:        about,
		HasAbout:     hasAbout,
	})
	if err != nil {
		return nil, profileErr(err)
	}
	r.pushUsernameUpdate(ctx, u)
	return r.tgSelfUser(u), nil
}

func (r *Router) onAccountCheckUsername(ctx context.Context, username string) (bool, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return false, internalErr()
	}
	svc, ok := r.deps.Users.(UserIdentityService)
	if !ok {
		return false, internalErr()
	}
	okUsername, err := svc.CheckUsername(ctx, userID, username)
	if err != nil {
		return false, usernameErr(err)
	}
	return okUsername, nil
}

func (r *Router) onAccountUpdateUsername(ctx context.Context, username string) (tg.UserClass, error) {
	userID, _, err := r.currentUserID(ctx)
	if err != nil {
		return nil, internalErr()
	}
	svc, ok := r.deps.Users.(UserIdentityService)
	if !ok {
		return nil, internalErr()
	}
	u, err := svc.UpdateUsername(ctx, userID, username)
	if err != nil {
		return nil, usernameErr(err)
	}
	r.pushUsernameUpdate(ctx, u)
	return r.tgSelfUser(u), nil
}

func (r *Router) pushUsernameUpdate(ctx context.Context, u domain.User) {
	if u.ID == 0 {
		return
	}
	r.pushUserUpdates(ctx, u.ID, &tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdateUserName{
			UserID:    u.ID,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Usernames: tgUsernames(u.Username),
		}},
		Users: []tg.UserClass{r.tgSelfUser(u)},
		Date:  int(r.clock.Now().Unix()),
	})
}

func tgAuthorization(a domain.Authorization, currentAuthKeyID [8]byte, now int) tg.Authorization {
	created := int(a.CreatedAt.Unix())
	if created == 0 {
		created = now
	}
	active := int(a.ActiveAt.Unix())
	if active == 0 {
		active = created
	}
	return tg.Authorization{
		Current:       a.AuthKeyID == currentAuthKeyID,
		OfficialApp:   true,
		Hash:          a.Hash,
		DeviceModel:   a.DeviceModel,
		Platform:      a.Platform,
		SystemVersion: a.SystemVersion,
		APIID:         a.APIID,
		AppName:       "Telegram Desktop",
		AppVersion:    a.AppVersion,
		DateCreated:   created,
		DateActive:    active,
		IP:            a.IP,
		Country:       "Unknown",
		Region:        "Unknown",
	}
}

func usernameErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrUsernameInvalid):
		return usernameInvalidErr()
	case errors.Is(err, domain.ErrUsernameOccupied):
		return usernameOccupiedErr()
	case errors.Is(err, domain.ErrUsernameNotOccupied):
		return usernameNotOccupiedErr()
	default:
		return internalErr()
	}
}

func profileErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrFirstNameInvalid):
		return firstNameInvalidErr()
	case errors.Is(err, domain.ErrAboutTooLong):
		return aboutTooLongErr()
	default:
		return internalErr()
	}
}
