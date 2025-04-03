package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/g0shi4ek/VK_bot/database"
	"github.com/g0shi4ek/VK_bot/internal/notifier"
	"github.com/g0shi4ek/VK_bot/internal/segmenter"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron      *cron.Cron
	db        *database.Database
	notifier  *notifier.Notifier
	segmenter *segmenter.Segmenter
}

func NewScheduler(db *database.Database, notifier *notifier.Notifier, segmenter *segmenter.Segmenter) *Scheduler {
	return &Scheduler{
		cron:      cron.New(),
		db:        db,
		notifier:  notifier,
		segmenter: segmenter,
	}
}

func (s *Scheduler) Start() {
	// фоновый процесс на отправку отложенных сообщений
	s.cron.AddFunc("@every 30s", s.processScheduledMailings)
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) processScheduledMailings() {
	ctx := context.Background()
	mailingRepo := database.NewMailingRepository(s.db)

	// Получение сообщений, которые должны быть отправлены сейчас
	mailings, err := mailingRepo.GetPendingMailings(ctx, time.Now())
	if err != nil {
		return
	}

	for _, mailing := range mailings {
		if err := s.notifier.SendMessageToSegment(mailing.Segment, mailing.Message); err != nil {
			log.Printf("Cannot send message")
		} else {
			mailing.IsSent = true
			log.Printf("send message")
			if err := mailingRepo.Update(ctx, mailing); err != nil {
				continue
			}
		}
	}
}
