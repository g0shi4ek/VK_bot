package segmenter

import (
	"context"
	"log"
	"time"

	"github.com/g0shi4ek/VK_bot/database"
	"github.com/g0shi4ek/VK_bot/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Segmenter struct {
	db *database.Database
}

func NewSegmenter(db *database.Database) *Segmenter {
	return &Segmenter{
		db: db,
	}
}

func (s *Segmenter) CreateSegmentIfNotExists(ctx context.Context, segmentName string) error {
	segmentRepo := database.NewSegmentRepository(s.db)

	seg, err := segmentRepo.GetByName(ctx, segmentName)
	if err != nil {
		return err
	}

	if seg != nil {
		return nil
	}

	//создаём
	segment := models.Segment{
		Name:      segmentName,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = segmentRepo.Create(ctx, &segment)
	log.Println("new segment")
	return err
}

func (s *Segmenter) AddUserToSegment(ctx context.Context, userID primitive.ObjectID, segment string) error {

	userRepo := database.NewUserRepository(s.db)
	user, err := userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Проверка что пользователь уже есть в сегменте
	for _, seg := range user.Segments {
		if seg == segment {
			return nil
		}
	}

	user.Segments = append(user.Segments, segment)
	return userRepo.Update(ctx, user)
}

func (s *Segmenter) RemoveUserFromSegment(ctx context.Context, userID primitive.ObjectID, segment string) error {
	segmentRepo := database.NewSegmentRepository(s.db)
	seg, err := segmentRepo.GetByName(ctx, segment)
	if err != nil {
		return err
	}
	if seg == nil {
		segmentRepo.Create(ctx, seg)
	}

	userRepo := database.NewUserRepository(s.db)
	user, err := userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Перезапись сегментов пользователя
	newSegments := make([]string, 0, len(user.Segments))
	for _, seg := range user.Segments {
		if seg != segment {
			newSegments = append(newSegments, seg)
		}
	}

	user.Segments = newSegments
	return userRepo.Update(ctx, user)
}

func (s *Segmenter) GetUsersInSegment(ctx context.Context, segment string) ([]*models.User, error) {
	userRepo := database.NewUserRepository(s.db)
	return userRepo.ListBySegment(ctx, segment)
}
