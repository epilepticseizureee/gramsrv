package bots

import (
	"context"
	"strings"
	"time"

	"telesrv/internal/domain"
)

const spamBotStartText = "This is a system bot created to ensure some features work correctly."

func (s *Service) respondAsSpamBot(userID int64, msg domain.Message) {
	mu := s.serviceBotReplyLock(domain.SpamBotUserID, userID)
	mu.Lock()
	defer mu.Unlock()

	text := strings.TrimSpace(msg.Body)
	if cmd, ok := parseBotCommand(text); !ok || cmd != "start" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s.sendServiceBotReply(ctx, domain.SpamBotUserID, userID, botReply{Text: spamBotStartText})
}
