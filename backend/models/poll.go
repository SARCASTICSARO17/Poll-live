package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a registered application user.
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
}

// Option represents a single answer choice in a poll.
type Option struct {
	ID   string `bson:"id" json:"id"`
	Text string `bson:"text" json:"text"`
}

// Poll represents a user-created poll.
type Poll struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Question  string             `bson:"question" json:"question"`
	Options   []Option           `bson:"options" json:"options"`
	CreatedBy primitive.ObjectID `bson:"createdBy" json:"createdBy"`
	Status    string             `bson:"status" json:"status"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
	ExpiresAt *time.Time         `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
}

// PollStatus constants.
const (
	PollStatusActive  = "active"
	PollStatusClosed  = "closed"
	PollStatusDeleted = "deleted"
)