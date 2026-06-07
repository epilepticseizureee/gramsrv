package privacy

import (
	"context"
	"testing"

	"telesrv/internal/domain"
	"telesrv/internal/store/memory"
)

func TestDefaultPrivacyRules(t *testing.T) {
	ctx := context.Background()
	svc := NewService(memory.NewPrivacyStore(), memory.NewContactStore())
	phone, err := svc.GetRules(ctx, 1001, domain.PrivacyKeyPhoneNumber)
	if err != nil {
		t.Fatalf("phone rules: %v", err)
	}
	if len(phone.Rules) != 1 || phone.Rules[0].Kind != domain.PrivacyRuleDisallowAll {
		t.Fatalf("phone default = %+v, want disallow all", phone.Rules)
	}
	birthday, err := svc.GetRules(ctx, 1001, domain.PrivacyKeyBirthday)
	if err != nil {
		t.Fatalf("birthday rules: %v", err)
	}
	if len(birthday.Rules) != 1 || birthday.Rules[0].Kind != domain.PrivacyRuleAllowContacts {
		t.Fatalf("birthday default = %+v, want allow contacts", birthday.Rules)
	}
	profile, err := svc.GetRules(ctx, 1001, domain.PrivacyKeyProfilePhoto)
	if err != nil {
		t.Fatalf("profile rules: %v", err)
	}
	if len(profile.Rules) != 1 || profile.Rules[0].Kind != domain.PrivacyRuleAllowAll {
		t.Fatalf("profile default = %+v, want allow all", profile.Rules)
	}
}

func TestAddAllowUserOverridesDisallowAll(t *testing.T) {
	ctx := context.Background()
	svc := NewService(memory.NewPrivacyStore(), memory.NewContactStore())
	if _, err := svc.SetRules(ctx, 1001, domain.PrivacyKeyPhoneNumber, []domain.PrivacyRule{{Kind: domain.PrivacyRuleDisallowAll}}); err != nil {
		t.Fatalf("set rules: %v", err)
	}
	allowed, err := svc.CanSee(ctx, 1001, 1002, domain.PrivacyKeyPhoneNumber)
	if err != nil {
		t.Fatalf("can see before: %v", err)
	}
	if allowed {
		t.Fatal("viewer should not see phone before exception")
	}
	if _, changed, err := svc.AddAllowUser(ctx, 1001, domain.PrivacyKeyPhoneNumber, 1002); err != nil {
		t.Fatalf("add allow: %v", err)
	} else if !changed {
		t.Fatal("first add allow should report changed")
	}
	allowed, err = svc.CanSee(ctx, 1001, 1002, domain.PrivacyKeyPhoneNumber)
	if err != nil {
		t.Fatalf("can see after: %v", err)
	}
	if !allowed {
		t.Fatal("viewer should see phone after allow-user exception")
	}
}

func TestExplicitDisallowUserWins(t *testing.T) {
	rules := domain.PrivacyRules{
		Key: domain.PrivacyKeyProfilePhoto,
		Rules: []domain.PrivacyRule{
			{Kind: domain.PrivacyRuleAllowAll},
			{Kind: domain.PrivacyRuleDisallowUsers, UserIDs: []int64{1002}},
		},
	}
	if Evaluate(rules, domain.PrivacyContext{OwnerUserID: 1001, ViewerUserID: 1002}) {
		t.Fatal("explicit disallow user should win over allow all")
	}
}
