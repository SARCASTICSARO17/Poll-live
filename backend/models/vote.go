package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VoterKey uniquely identifies a voter for duplicate-vote prevention.
// For authenticated users it is "user:<id>"; for anonymous visitors it is
// "anon:<browserVoterId>".
type VoterKey string

// Vote represents a single cast vote stored permanently in MongoDB.
type Vote struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PollID    primitive.ObjectID `bson:"pollId" json:"pollId"`
	OptionID  string             `bson:"optionId" json:"optionId"`
	VoterKey  string             `bson:"voterKey" json:"-"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

// NewVoterKey builds a voter identifier string from a raw voter id and whether
// that voter is an authenticated user.
func NewVoterKey(voterID string, authenticated bool) string {
	if authenticated {
		return "user:" + voterID
	}
	return "anon:" + voterID
}