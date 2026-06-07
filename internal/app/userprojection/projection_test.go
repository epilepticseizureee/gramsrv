package userprojection

import (
	"context"
	"testing"

	privacyapp "telesrv/internal/app/privacy"
	"telesrv/internal/domain"
	"telesrv/internal/store/memory"
)

func TestProjectorCombinesProfilePhotosAndViewerContacts(t *testing.T) {
	ctx := context.Background()
	const viewerID int64 = 1001
	const friendID int64 = 1002
	const strangerID int64 = 1003
	contacts := memory.NewContactStore()
	if _, err := contacts.Upsert(ctx, viewerID, domain.ContactInput{
		ContactUserID: friendID,
		Phone:         "1111",
		FirstName:     "Alice",
		LastName:      "Contact",
	}); err != nil {
		t.Fatalf("upsert contact: %v", err)
	}
	projector := New(
		WithContactStore(contacts),
		WithPhotoProvider(fakeProfilePhotos{
			profile: map[int64]domain.ProfilePhotoRef{
				friendID:   {PhotoID: 9001, DCID: 2, Stripped: []byte{1, 2}},
				strangerID: {PhotoID: 9002, DCID: 3, Stripped: []byte{3, 4}},
			},
		}),
	)

	users, err := projector.ForViewer(ctx, viewerID, []domain.User{
		{ID: viewerID, Phone: "15550000001", FirstName: "Owner"},
		{ID: friendID, AccessHash: 22, Phone: "15550000002", FirstName: "Public", LastName: "Name"},
		{ID: strangerID, AccessHash: 33, Phone: "15550000003", FirstName: "Stranger"},
	})
	if err != nil {
		t.Fatalf("ForViewer: %v", err)
	}

	friend := projectionUser(t, users, friendID)
	if friend.FirstName != "Alice" || friend.LastName != "Contact" || friend.Phone != "1111" || !friend.Contact {
		t.Fatalf("friend projection = %+v, want contact name/phone", friend)
	}
	if friend.PhotoID != 9001 || friend.PhotoDCID != 2 || string(friend.PhotoStripped) != string([]byte{1, 2}) {
		t.Fatalf("friend photo = id %d dc %d stripped %v, want 9001/2/[1 2]", friend.PhotoID, friend.PhotoDCID, friend.PhotoStripped)
	}
	stranger := projectionUser(t, users, strangerID)
	if stranger.Phone != "" || stranger.Contact {
		t.Fatalf("stranger projection = %+v, want hidden phone and non-contact", stranger)
	}
	if stranger.PhotoID != 9002 || stranger.PhotoDCID != 3 {
		t.Fatalf("stranger photo = id %d dc %d, want 9002/3", stranger.PhotoID, stranger.PhotoDCID)
	}
}

func TestProjectorPersonalPhotoWinsOverProfile(t *testing.T) {
	ctx := context.Background()
	const viewerID int64 = 2001
	const friendID int64 = 2002
	contacts := memory.NewContactStore()
	if _, err := contacts.Upsert(ctx, viewerID, domain.ContactInput{ContactUserID: friendID, FirstName: "Friend"}); err != nil {
		t.Fatalf("upsert contact: %v", err)
	}
	if _, _, err := contacts.SetPersonalPhoto(ctx, viewerID, friendID, 9100, 100); err != nil {
		t.Fatalf("set personal photo: %v", err)
	}
	projector := New(
		WithContactStore(contacts),
		WithPhotoProvider(fakeProfilePhotos{
			profile: map[int64]domain.ProfilePhotoRef{friendID: {PhotoID: 9001, DCID: 2}},
		}),
	)
	users, err := projector.ForViewer(ctx, viewerID, []domain.User{{ID: friendID, FirstName: "Public"}})
	if err != nil {
		t.Fatalf("ForViewer: %v", err)
	}
	friend := projectionUser(t, users, friendID)
	if friend.PhotoID != 9100 || !friend.PhotoPersonal {
		t.Fatalf("friend photo = id %d personal %v, want personal 9100", friend.PhotoID, friend.PhotoPersonal)
	}
}

func TestProjectorUsesFallbackWhenProfilePhotoHidden(t *testing.T) {
	ctx := context.Background()
	const viewerID int64 = 3001
	const ownerID int64 = 3002
	contacts := memory.NewContactStore()
	rules := memory.NewPrivacyStore()
	privacy := privacyapp.NewService(rules, contacts)
	if _, err := privacy.SetRules(ctx, ownerID, domain.PrivacyKeyProfilePhoto, []domain.PrivacyRule{{Kind: domain.PrivacyRuleDisallowAll}}); err != nil {
		t.Fatalf("set privacy: %v", err)
	}
	projector := New(
		WithContactStore(contacts),
		WithPrivacyEvaluator(privacy),
		WithPhotoProvider(fakeProfilePhotos{
			profile:  map[int64]domain.ProfilePhotoRef{ownerID: {PhotoID: 9001, DCID: 2}},
			fallback: map[int64]domain.ProfilePhotoRef{ownerID: {PhotoID: 9002, DCID: 3}},
		}),
	)
	users, err := projector.ForViewer(ctx, viewerID, []domain.User{{ID: ownerID, Phone: "15550003002", FirstName: "Owner"}})
	if err != nil {
		t.Fatalf("ForViewer: %v", err)
	}
	owner := projectionUser(t, users, ownerID)
	if owner.PhotoID != 9002 || owner.PhotoDCID != 3 || owner.Phone != "" {
		t.Fatalf("owner projection = %+v, want fallback photo and hidden phone", owner)
	}
}

func projectionUser(t *testing.T, users []domain.User, id int64) domain.User {
	t.Helper()
	for _, user := range users {
		if user.ID == id {
			return user
		}
	}
	t.Fatalf("user %d not found in %+v", id, users)
	return domain.User{}
}

type fakeProfilePhotos struct {
	profile  map[int64]domain.ProfilePhotoRef
	fallback map[int64]domain.ProfilePhotoRef
}

func (p fakeProfilePhotos) CurrentProfilePhotos(_ context.Context, _ domain.PeerType, ids []int64) (map[int64]domain.ProfilePhotoRef, error) {
	return p.CurrentProfilePhotosKind(context.Background(), domain.PeerTypeUser, ids, domain.ProfilePhotoKindProfile)
}

func (p fakeProfilePhotos) CurrentProfilePhotosKind(_ context.Context, _ domain.PeerType, ids []int64, kind domain.ProfilePhotoKind) (map[int64]domain.ProfilePhotoRef, error) {
	source := p.profile
	if kind == domain.ProfilePhotoKindFallback {
		source = p.fallback
	}
	out := make(map[int64]domain.ProfilePhotoRef, len(ids))
	for _, id := range ids {
		if ref, ok := source[id]; ok {
			out[id] = ref
		}
	}
	return out, nil
}
