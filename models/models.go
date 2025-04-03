package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	ChatID    string             `bson:"chat_id"`
	FirstName string             `bson:"first_name"`
	LastName  string             `bson:"last_name"`
	Segments  []string           `bson:"segments"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

type Mailing struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `bson:"name"`
	Message     string             `bson:"message"`
	Segment     string             `bson:"segment"`
	ScheduledAt time.Time          `bson:"scheduled_at"`
	IsSent      bool               `bson:"is_sent"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}

type Segment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `bson:"name"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}