package database

import (
	"context"
	"time"

	"github.com/g0shi4ek/VK_bot/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *Database) *UserRepository {
	return &UserRepository{
		collection: db.GetCollection("users"),
	}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.CreatedAt = time.Now().UTC().Truncate(time.Minute)
	user.UpdatedAt = time.Now().UTC().Truncate(time.Minute)

	_, err := r.collection.InsertOne(ctx, user)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByChatID(ctx context.Context, chatID string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now().UTC().Truncate(time.Minute)

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": user.ID},
		bson.M{"$set": user},
	)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *UserRepository) ListBySegment(ctx context.Context, segment string) ([]*models.User, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"segments": segment})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}
