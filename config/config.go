package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken string
	DatabaseURL string
	DatabaseName string
	Debug bool
}

func LoadConfig() (*Config, error) {
	//Загрузка .env файла
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cfg := &Config{
		BotToken: os.Getenv("BOT_TOKEN"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		DatabaseName: os.Getenv("DATABASE_NAME"),
		Debug: os.Getenv("DEBUG") == "true",
	}

	if cfg.BotToken == "" {
		return nil, errors.New("bot token is required")

	}

	return cfg, nil
}
