package bot

import (
	"context"
	"log"

	"github.com/g0shi4ek/VK_bot/database"
	"github.com/g0shi4ek/VK_bot/models"
	"github.com/mail-ru-im/bot-golang"
)

// Middleware
func (h *Handler) withLogging(next func(*botgolang.Message, []string)) func(*botgolang.Message, []string) {
	return func(msg *botgolang.Message, args []string) {
		log.Printf("Received command from %s: %s", msg.Chat.ID, msg.Text)
		next(msg, args)
	}
}

func (h *Handler) withAuth(next func(*botgolang.Message, *models.User, []string)) func(*botgolang.Message, []string) {
	return func(msg *botgolang.Message, args []string) {
		userRepo := database.NewUserRepository(h.db)
		user, err := userRepo.GetByChatID(context.Background(), msg.Chat.ID)
		if err != nil || user == nil {
			h.notifier.SendMessage(msg.Chat.ID, "Пожалуйста, сначала зарегистрируйтесь с помощью команды /start")
			return
		}
		next(msg, user, args)
	}
}