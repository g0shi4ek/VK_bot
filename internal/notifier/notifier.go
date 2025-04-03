package notifier

import (
	"context"
	"log"

	"github.com/g0shi4ek/VK_bot/database"
	botgolang "github.com/mail-ru-im/bot-golang"
)

type Notifier struct {
	bot *botgolang.Bot
	db  *database.Database
}

func NewNotifier(bot *botgolang.Bot, db *database.Database) *Notifier {
	return &Notifier{
		bot: bot,
		db:  db,
	}
}

func (n *Notifier) SendMessage(chatID, text string) error {
	message := n.bot.NewTextMessage(chatID, text)
	err := message.Send()
	if err != nil {
		log.Printf("Failed to send message to chat %s: %v", chatID, err)
		return err
	}

	log.Printf("Message sent to chat %s: %s", chatID, text)
	return nil
}

func (n *Notifier) SendMessageToSegment(segment, text string) error {
	// Получение пользователей по сегменту
	log.Println("aaaaaaaaaaaaaaaaa")
	ctx := context.Background()
	userRepo := database.NewUserRepository(n.db)
	users, err := userRepo.ListBySegment(ctx, segment)
	if err != nil {
		return err
	}

	// Отправка пользователю
	for _, user := range users {
		if err := n.SendMessage(user.ChatID, text); err != nil {
			log.Printf("Failed to send message to user %s: %v", user.ChatID, err)
			continue
		}
	}

	return nil
}
