package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/g0shi4ek/VK_bot/config"
	"github.com/g0shi4ek/VK_bot/database"
	"github.com/g0shi4ek/VK_bot/internal/bot"
	"github.com/g0shi4ek/VK_bot/internal/notifier"
	"github.com/g0shi4ek/VK_bot/internal/scheduler"
	"github.com/g0shi4ek/VK_bot/internal/segmenter"
	botgolang "github.com/mail-ru-im/bot-golang"
)

func main() {
	// загрузка конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Mongodb клиент
	dbClient, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbClient.Disconnect(context.Background())

	// инициализация бота
	vkBot, _ := botgolang.NewBot(cfg.BotToken, botgolang.BotDebug(cfg.Debug))

	// инициализация сервисов
	segmenterService := segmenter.NewSegmenter(dbClient)
	notifierService := notifier.NewNotifier(vkBot, dbClient)
	schedulerService := scheduler.NewScheduler(dbClient, notifierService, segmenterService)
	// заполнение базовых сегментов
	baseSegments := []string{"all", "clients", "workers"}
	for _, name := range baseSegments {
		err := segmenterService.CreateSegmentIfNotExists(context.Background(), name)
		if err != nil {
			log.Printf("Ошибка инициализации сегмента %s: %v", name, err)
		}
	}

	// подключение хендлеров
	botHandler := bot.NewHandler(vkBot, dbClient, notifierService, segmenterService, schedulerService)

	// отложенные
	schedulerService.Start()

	// запуск бота
	go func() {
		if err := botHandler.Start(); err != nil {
			log.Fatalf("Failed to start bot: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// остановка отложенных
	schedulerService.Stop()

	// отключение от бд
	if err := dbClient.Disconnect(ctx); err != nil {
		log.Fatalf("Failed to disconnect from database: %v", err)
	}

	log.Println("Server exited properly")
}
