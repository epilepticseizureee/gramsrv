package bots

import (
	"context"
	"testing"

	"telesrv/internal/domain"
	"telesrv/internal/store/memory"
)

func latestSpamBotReply(t *testing.T, messages *memory.MessageStore, userID int64) domain.Message {
	t.Helper()
	list, err := messages.ListByUser(context.Background(), userID, domain.MessageFilter{
		HasPeer: true,
		Peer:    domain.Peer{Type: domain.PeerTypeUser, ID: domain.SpamBotUserID},
		Limit:   100,
	})
	if err != nil {
		t.Fatalf("list spambot history: %v", err)
	}
	var latest domain.Message
	for _, msg := range list.Messages {
		if msg.From.ID == domain.SpamBotUserID && msg.ID > latest.ID {
			latest = msg
		}
	}
	if latest.ID == 0 {
		t.Fatal("no SpamBot reply")
	}
	return latest
}

func TestSpamBotSystemSeedAndStart(t *testing.T) {
	svc, users, bots, messages := newTestService(t)
	owner := newOwner(t, users, "+4200")
	ctx := context.Background()

	if !svc.HandlesBot(domain.SpamBotUserID) {
		t.Fatal("service should handle SpamBot")
	}
	u, found, err := users.ByUsername(ctx, "SpamBot")
	if err != nil || !found {
		t.Fatalf("@SpamBot user not seeded: found=%v err=%v", found, err)
	}
	if u.ID != domain.SpamBotUserID || u.FirstName != "Spam Info Bot" || u.Username != "SpamBot" ||
		!u.Verified || !u.Bot || u.BotInfoVersion < 1 {
		t.Fatalf("@SpamBot user = %+v, want verified seeded bot", u)
	}
	profile, found, err := bots.GetBot(ctx, domain.SpamBotUserID)
	if err != nil || !found {
		t.Fatalf("@SpamBot profile not seeded: found=%v err=%v", found, err)
	}
	if profile.Description != "The official Spam Info Bot. Helps users with limited accounts regain the full functionality." {
		t.Fatalf("@SpamBot description = %q", profile.Description)
	}
	if len(profile.Commands) != 0 {
		t.Fatalf("@SpamBot commands = %+v, want empty list", profile.Commands)
	}

	svc.respondAsSpamBot(owner.ID, domain.Message{
		From: domain.Peer{Type: domain.PeerTypeUser, ID: owner.ID},
		Peer: domain.Peer{Type: domain.PeerTypeUser, ID: domain.SpamBotUserID},
		Body: "/start",
	})
	if reply := latestSpamBotReply(t, messages, owner.ID); reply.Body != spamBotStartText {
		t.Fatalf("@SpamBot /start reply = %q, want %q", reply.Body, spamBotStartText)
	}
}
