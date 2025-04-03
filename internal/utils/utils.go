package utils

import (
	"fmt"
	"log"
	"strings"
	"time"
)

func ParseTime(input string) (time.Time, error) {
	// Время в разных форматах
	formats := []string{
		"02.01.2006 15:04",
		"02.01.2006",
		"2006-01-02 15:04",
		"2006-01-02",
		time.RFC3339,
		"15:04 02.01.2006",
	}

	// Указываем московский часовой пояс
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.Time{}, err
	}

	for _, format := range formats {
		// Парсим с учётом MSK
		t, err := time.ParseInLocation(format, input, loc)
		if err == nil {
			log.Println("Time: t", t)
			return t, nil // Возвращаем время в MSK
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", input)
}

func ParseCommand(text string) (string, []string) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "", nil
	}

	command := strings.TrimPrefix(parts[0], "/")
	if len(parts) > 1 {
		return command, parts[1:]
	}
	log.Print(command)
	return command, nil
}
