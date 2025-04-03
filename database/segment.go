package database

import (
	"context"
	"time"

	"github.com/g0shi4ek/VK_bot/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SegmentRepository struct {
	collection *mongo.Collection
}

func NewSegmentRepository(db *Database) *SegmentRepository {
	return &SegmentRepository{
		collection: db.GetCollection("segments"),
	}
}

func (r *SegmentRepository) Create(ctx context.Context, segment *models.Segment) error {
	segment.CreatedAt = time.Now().UTC().Truncate(time.Minute)
	segment.UpdatedAt = time.Now().UTC().Truncate(time.Minute)

	_, err := r.collection.InsertOne(ctx, segment)
	return err
}

func (r *SegmentRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Segment, error) {
	var segment models.Segment
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&segment)
	if err != nil {
		return nil, err
	}
	return &segment, nil
}

func (r *SegmentRepository) GetByName(ctx context.Context, name string) (*models.Segment, error) {
	var segment models.Segment
	err := r.collection.FindOne(ctx, bson.M{"name": name}).Decode(&segment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &segment, nil
}

func (r *SegmentRepository) ListAll(ctx context.Context) ([]*models.Segment, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var segments []*models.Segment
	if err := cursor.All(ctx, &segments); err != nil {
		return nil, err
	}
	return segments, nil
}

func (r *SegmentRepository) Update(ctx context.Context, segment *models.Segment) error {
	segment.UpdatedAt = time.Now().UTC().Truncate(time.Minute)

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": segment.ID},
		bson.M{"$set": segment},
	)
	return err
}

func (r *SegmentRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
