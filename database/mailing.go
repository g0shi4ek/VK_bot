package database

import (
	"context"
	"log"
	"time"

	"github.com/g0shi4ek/VK_bot/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MailingRepository struct {
	collection *mongo.Collection
}

func NewMailingRepository(db *Database) *MailingRepository {
	return &MailingRepository{
		collection: db.GetCollection("mailings"),
	}
}

func (r *MailingRepository) Create(ctx context.Context, mailing *models.Mailing) error {
	//mailing.CreatedAt = time.Now().UTC().Truncate(time.Minute)
	mailing.UpdatedAt = time.Now().UTC().Truncate(time.Minute)

	_, err := r.collection.InsertOne(ctx, mailing)
	return err
}

func (r *MailingRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Mailing, error) {
	var mailing models.Mailing
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&mailing)
	if err != nil {
		return nil, err
	}
	return &mailing, nil
}

func (r *MailingRepository) Update(ctx context.Context, mailing *models.Mailing) error {
	mailing.UpdatedAt = time.Now().UTC().Truncate(time.Minute)

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": mailing.ID},
		bson.M{"$set": mailing},
	)
	return err
}

func (r *MailingRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *MailingRepository) GetPendingMailings(ctx context.Context, now time.Time) ([]*models.Mailing, error) {
	log.Println("check all before: ", now.UTC().Truncate(time.Minute))

	cursor, err := r.collection.Find(ctx, bson.M{
		"scheduled_at": bson.M{
			"$lte": now.UTC().Truncate(time.Minute), // Все, чьё время уже наступило
		},
		"is_sent": false,
	})
	if err != nil {
		log.Println("ERROR")
		return nil, err
	}
	defer cursor.Close(ctx)

	var mailings []*models.Mailing
	if err := cursor.All(ctx, &mailings); err != nil {
		log.Println("ERROR")
		return nil, err
	}
	if len(mailings) == 0 {
		log.Println("0 mailings")
		return nil, nil
	}
	return mailings, nil
}

func (r *MailingRepository) ListAll(ctx context.Context) ([]*models.Mailing, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var mailings []*models.Mailing
	if err := cursor.All(ctx, &mailings); err != nil {
		return nil, err
	}
	return mailings, nil
}
