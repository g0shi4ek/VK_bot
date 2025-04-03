package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/g0shi4ek/VK_bot/database"
	"github.com/g0shi4ek/VK_bot/internal/notifier"
	"github.com/g0shi4ek/VK_bot/internal/scheduler"
	"github.com/g0shi4ek/VK_bot/internal/segmenter"
	"github.com/g0shi4ek/VK_bot/internal/utils"
	"github.com/g0shi4ek/VK_bot/models"
	botgolang "github.com/mail-ru-im/bot-golang"
)

type Handler struct {
	bot           *botgolang.Bot
	db            *database.Database
	notifier      *notifier.Notifier
	segmenter     *segmenter.Segmenter
	scheduler     *scheduler.Scheduler
	commandRouter map[string]func(*botgolang.Message, []string)
	userStates    map[string]UserState
	mu            sync.Mutex
}

type UserState struct {
	Status string
	Data   map[string]interface{}
}

func NewHandler(bot *botgolang.Bot, db *database.Database, notifier *notifier.Notifier,
	segmenter *segmenter.Segmenter, scheduler *scheduler.Scheduler) *Handler {
	h := &Handler{
		bot:        bot,
		db:         db,
		notifier:   notifier,
		segmenter:  segmenter,
		scheduler:  scheduler,
		userStates: make(map[string]UserState),
	}

	h.commandRouter = map[string]func(*botgolang.Message, []string){
		"start":          h.handleStart,
		"help":           h.withLogging(h.handleHelp),
		"create_mailing": h.withLogging(h.withAuth(h.handleCreateMailing)),
		"list_mailings":  h.withLogging(h.withAuth(h.handleListMailings)),
		"add_segment":    h.withLogging(h.withAuth(h.handleAddSegment)),
		"remove_segment": h.withLogging(h.withAuth(h.handleRemoveSegment)),
		"list_segments":  h.withLogging(h.withAuth(h.handleListSegments)),
		"cancel":         h.withLogging(h.withAuth(h.handleCancel)),
	}

	return h
}

func (h *Handler) Start() error {
	log.Println("Starting bot handler...")
	updates := h.bot.GetUpdatesChannel(context.Background())

	for update := range updates {
		if update.Type == botgolang.NEW_MESSAGE {
			msg := update.Payload.Message()
			fmt.Println(msg)

			// Обработка состояния пользователя
			if h.checkUserState(msg) {
				continue
			}

			// Обработка команд
			if msg.Text != "" && msg.Text[0] == '/' {
				command, args := utils.ParseCommand(msg.Text)
				if handler, ok := h.commandRouter[command]; ok {
					go handler(msg, args)
					continue
				}
			}

			// Обработка обычных сообщений
			go h.handleMessage(msg)
		}
	}

	return nil
}

// команда /start
func (h *Handler) handleStart(msg *botgolang.Message, args []string) {
	userRepo := database.NewUserRepository(h.db)
	user, _ := userRepo.GetByChatID(context.Background(), msg.Chat.ID)
	if user != nil {
		h.notifier.SendMessage(msg.Chat.ID, "Вы уже зарегистрированы! Используйте /help для списка команд.")
		return
	}

	from := msg.Chat
	// Создание нового пользователя
	newUser := &models.User{
		ChatID:    from.ID,
		FirstName: from.FirstName,
		LastName:  from.LastName,
		Segments:  []string{"all"},
	}

	err := userRepo.Create(context.Background(), newUser)
	if err != nil {
		h.notifier.SendMessage(msg.Chat.ID, "Ошибка при регистрации. Пожалуйста, попробуйте снова.")
		return
	}

	h.notifier.SendMessage(msg.Chat.ID,
		"Добро пожаловать! Вы успешно зарегистрированы.\n\n"+
			"Используйте /help для списка доступных команд.")
}

// команда /help
func (h *Handler) handleHelp(msg *botgolang.Message, args []string) {
	helpText := `📋 Доступные команды:

/start - Регистрация в системе
/help - Показать это сообщение

📬 Работа с рассылками:
/create_mailing - Создать новую рассылку
/list_mailings - Список всех рассылок

🏷️ Работа с сегментами:
/add_segment - Добавить пользователя в сегмент
/remove_segment - Удалить пользователя из сегмента
/list_segments - Список всех сегментов

❌ /cancel - Отменить текущее действие`

	h.notifier.SendMessage(msg.Chat.ID, helpText)
}

// /create_mailing
func (h *Handler) handleCreateMailing(msg *botgolang.Message, user *models.User, args []string) {
	// Многошаговая команда - сохраняем состояние
	st := make(map[string]interface{}, 0)
	h.saveUserState(msg.Chat.ID, "awaiting_mailing_name", st)

	h.notifier.SendMessage(msg.Chat.ID,
		"Создание новой рассылки. Ответьте на несколько вопросов:\n\n"+
			"1. Введите название рассылки:")
}

// /list_mailings
func (h *Handler) handleListMailings(msg *botgolang.Message, user *models.User, args []string) {
	mailingRepo := database.NewMailingRepository(h.db)
	mailings, err := mailingRepo.ListAll(context.Background())
	if err != nil {
		h.notifier.SendMessage(msg.Chat.ID, "Ошибка при получении списка рассылок.")
		return
	}

	if len(mailings) == 0 {
		h.notifier.SendMessage(msg.Chat.ID, "Нет активных рассылок.")
		return
	}

	var response strings.Builder
	response.WriteString("📫 Список рассылок:\n\n")
	for _, mailing := range mailings {
		status := "🟢 Активна"
		if mailing.IsSent {
			status = "✅ Отправлена"
		}
		response.WriteString(fmt.Sprintf(
			"%s\n"+
				"Сегмент: %s\n"+
				"Дата: %s\n"+
				"Статус: %s\n\n",
			mailing.Name,
			mailing.Segment,
			mailing.ScheduledAt.Format("02.01.2006 15:04"),
			status,
		))
	}

	h.notifier.SendMessage(msg.Chat.ID, response.String())
}

// /add_segment
func (h *Handler) handleAddSegment(msg *botgolang.Message, user *models.User, args []string) {
	if len(args) == 0 {
		// список доступных сегментов
		segmentRepo := database.NewSegmentRepository(h.db)
		segments, err := segmentRepo.ListAll(context.Background())
		if err != nil {
			h.notifier.SendMessage(msg.Chat.ID, "Ошибка при получении списка сегментов.")
			return
		}

		var response strings.Builder
		response.WriteString("🏷️ Доступные сегменты:\n\n")
		for _, segment := range segments {
			response.WriteString(fmt.Sprintf(
				" - %s\n",
				segment.Name,
			))
		}
		response.WriteString("Используйте: /add_segment [название_сегмента]")

		h.notifier.SendMessage(msg.Chat.ID, response.String())
		return
	}

	segmentName := args[0]

	//создаём сегмент (если не существует)
	if err := h.segmenter.CreateSegmentIfNotExists(context.Background(), segmentName); err != nil {
		log.Printf("Ошибка создания сегмента: %v", err)
		h.notifier.SendMessage(msg.Chat.ID, "Ошибка при обработке сегмента")
		return
	}

	err := h.segmenter.AddUserToSegment(context.Background(), user.ID, segmentName)
	if err != nil {
		h.notifier.SendMessage(msg.Chat.ID, "Ошибка при добавлении в сегмент.")
		return
	}

	h.notifier.SendMessage(msg.Chat.ID,
		fmt.Sprintf("Вы успешно добавлены в сегмент %s!", segmentName))
}

// /remove_segment
func (h *Handler) handleRemoveSegment(msg *botgolang.Message, user *models.User, args []string) {
	if len(args) == 0 {
		// сегменты пользователя
		var response strings.Builder
		response.WriteString("🏷️ Ваши сегменты:\n\n")
		for _, segment := range user.Segments {
			response.WriteString(fmt.Sprintf("- %s\n", segment))
		}
		response.WriteString("\nИспользуйте: /remove_segment [название_сегмента]")

		h.notifier.SendMessage(msg.Chat.ID, response.String())
		return
	}

	segmentName := args[0]
	err := h.segmenter.RemoveUserFromSegment(context.Background(), user.ID, segmentName)
	if err != nil {
		h.notifier.SendMessage(msg.Chat.ID, "Ошибка при удалении из сегмента.")
		return
	}

	h.notifier.SendMessage(msg.Chat.ID,
		fmt.Sprintf("Вы успешно удалены из сегмента %s!", segmentName))
}

// /list_segments
func (h *Handler) handleListSegments(msg *botgolang.Message, user *models.User, args []string) {
	segmentRepo := database.NewSegmentRepository(h.db)
	segments, err := segmentRepo.ListAll(context.Background())
	if err != nil {
		h.notifier.SendMessage(msg.Chat.ID, "Ошибка при получении списка сегментов.")
		return
	}

	var response strings.Builder
	response.WriteString("🏷️ Все сегменты:\n\n")
	for _, segment := range segments {
		// Проверяем, состоит ли пользователь в этом сегменте
		inSegment := false
		for _, userSegment := range user.Segments {
			if userSegment == segment.Name {
				inSegment = true
				break
			}
		}

		status := "❌ Не входите"
		if inSegment {
			status = "✅ Входите"
		}

		response.WriteString(fmt.Sprintf(
			"%s\n%s\n\n",
			segment.Name,
			status,
		))
	}

	h.notifier.SendMessage(msg.Chat.ID, response.String())
}

// /cancel
func (h *Handler) handleCancel(msg *botgolang.Message, user *models.User, args []string) {
	h.clearUserState(msg.Chat.ID)
	h.notifier.SendMessage(msg.Chat.ID, "Текущее действие отменено.")
}

// обрабатывает обычные сообщения (не команды)
func (h *Handler) handleMessage(msg *botgolang.Message) {
	// Проверяем, есть ли у пользователя активное состояние
	if state, exists := h.getUserState(msg.Chat.ID); exists {
		switch state.Status {
		case "awaiting_mailing_name":
			h.processMailingName(msg, state)
		case "awaiting_mailing_segment":
			h.processMailingSegment(msg, state)
		case "awaiting_mailing_date":
			h.processMailingDate(msg, state)
		case "awaiting_mailing_message":
			h.processMailingMessage(msg, state)
		default:
			h.notifier.SendMessage(msg.Chat.ID, "Неизвестное состояние. Используйте /cancel для отмены.")
		}
		return
	}

	// Обработка обычных сообщений
	h.notifier.SendMessage(msg.Chat.ID,
		"Я не понимаю ваше сообщение. Используйте /help для списка команд.")
}

// брабатывает название рассылки (шаг 1)
func (h *Handler) processMailingName(msg *botgolang.Message, state UserState) {
	// Сохраняем название и переходим к следующему шагу
	state.Data["name"] = msg.Text
	state.Status = "awaiting_mailing_segment"
	h.saveUserState(msg.Chat.ID, state.Status, state.Data)

	h.notifier.SendMessage(msg.Chat.ID,
		"2. Укажите сегмент для рассылки (или 'all' для всех пользователей):")
}

// обрабатывает сегмент рассылки (шаг 2)
func (h *Handler) processMailingSegment(msg *botgolang.Message, state UserState) {
	// Проверяем существование сегмента
	if msg.Text != "all" {
		segmentRepo := database.NewSegmentRepository(h.db)
		_, err := segmentRepo.GetByName(context.Background(), msg.Text)
		if err != nil {
			h.notifier.SendMessage(msg.Chat.ID,
				"Сегмент не найден. Укажите существующий сегмент или 'all'.")
			return
		}
	}

	state.Data["segment"] = msg.Text
	state.Status = "awaiting_mailing_date"
	h.saveUserState(msg.Chat.ID, state.Status, state.Data)

	h.notifier.SendMessage(msg.Chat.ID,
		"3. Укажите дату и время рассылки (например: 31.12.2023 23:59):")
}

// обрабатывает дату рассылки (шаг 3)
func (h *Handler) processMailingDate(msg *botgolang.Message, state UserState) {
	// Парсим дату
	scheduledAt, err := utils.ParseTime(msg.Text)
	if err != nil {
		h.notifier.SendMessage(msg.Chat.ID,
			"Неверный формат даты. Укажите в формате ДД.ММ.ГГГГ ЧЧ:ММ")
		return
	}

	// Проверяем, что дата в будущем
	if scheduledAt.Before(time.Now()) {
		h.notifier.SendMessage(msg.Chat.ID,
			"Дата должна быть в будущем. Укажите корректную дату.")
		return
	}

	state.Data["scheduled_at"] = scheduledAt.UTC()
	log.Println("scheduled_at", state.Data["scheduled_at"])
	state.Status = "awaiting_mailing_message"
	h.saveUserState(msg.Chat.ID, state.Status, state.Data)

	h.notifier.SendMessage(msg.Chat.ID,
		"4. Введите текст сообщения для рассылки:")
}

// обрабатывает текст рассылки (шаг 4)
func (h *Handler) processMailingMessage(msg *botgolang.Message, state UserState) {
	// Создаем рассылку
	mailing := &models.Mailing{
		Name:        state.Data["name"].(string),
		Segment:     state.Data["segment"].(string),
		Message:     msg.Text,
		ScheduledAt: state.Data["scheduled_at"].(time.Time),
		IsSent:      false,
	}

	log.Println("created with", mailing.ScheduledAt)

	mailingRepo := database.NewMailingRepository(h.db)
	err := mailingRepo.Create(context.Background(), mailing)
	if err != nil {
		h.notifier.SendMessage(msg.Chat.ID, "Ошибка при создании рассылки.")
		return
	}

	// Очищаем состояние
	h.clearUserState(msg.Chat.ID)

	mskLoc, _ := time.LoadLocation("Europe/Moscow")
	mskTime := mailing.ScheduledAt.In(mskLoc)
	h.notifier.SendMessage(msg.Chat.ID,
		fmt.Sprintf("✅ Рассылка %s успешно создана!\n\n"+
			"Сегмент: %s\n"+
			"Дата отправки: %s",
			mailing.Name,
			mailing.Segment,
			mskTime.Format("02.01.2006 15:04")))
}

// методы для работы с состояниями пользователей

func (h *Handler) saveUserState(chatID string, status string, data map[string]interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.userStates[chatID] = UserState{
		Status: status,
		Data:   data,
	}
}

func (h *Handler) getUserState(chatID string) (UserState, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	state, ok := h.userStates[chatID]
	return state, ok
}

func (h *Handler) clearUserState(chatID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.userStates, chatID)
}

func (h *Handler) checkUserState(msg *botgolang.Message) bool {
	if state, exists := h.getUserState(msg.Chat.ID); exists {
		switch state.Status {
		case "awaiting_mailing_name":
			h.processMailingName(msg, state)
		case "awaiting_mailing_segment":
			h.processMailingSegment(msg, state)
		case "awaiting_mailing_date":
			h.processMailingDate(msg, state)
		case "awaiting_mailing_message":
			h.processMailingMessage(msg, state)
		default:
			h.notifier.SendMessage(msg.Chat.ID, "Неизвестное состояние. Используйте /cancel для отмены.")
		}
		return true
	}
	return false
}
