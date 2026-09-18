package repositories

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"poll-live/backend/models"
)

var ErrVoteAlreadyExists = errors.New("you have already voted in this poll")
var ErrVoteNotFound = errors.New("vote not found")

// VoteRepository provides data access for votes.
type VoteRepository struct {
	collection *mongo.Collection
}

// NewVoteRepository creates a vote repository backed by MongoDB.
func NewVoteRepository(db *mongo.Database) *VoteRepository {
	return &VoteRepository{collection: db.Collection("votes")}
}

// Create inserts a new vote. The unique index on (pollId, voterKey) enforces
// duplicate-vote prevention atomically, even under concurrent requests.
func (r *VoteRepository) Create(ctx context.Context, vote *models.Vote) error {
	vote.ID = primitive.NewObjectID()
	vote.CreatedAt = now()
	_, err := r.collection.InsertOne(ctx, vote)

	var mongoErr mongo.WriteException
	if errors.As(err, &mongoErr) {
		for _, we := range mongoErr.WriteErrors {
			if we.Code == 11000 {
				return ErrVoteAlreadyExists
			}
		}
	}
	return err
}

// ExistsByVoterKey reports whether a voter has already voted in a poll.
func (r *VoteRepository) ExistsByVoterKey(ctx context.Context, pollID primitive.ObjectID, voterKey string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"pollId": pollID, "voterKey": voterKey})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountByOption returns the number of votes per option for a poll. It is used
// to seed Redis counters when a poll's results have not yet been cached.
func (r *VoteRepository) CountByOption(ctx context.Context, pollID primitive.ObjectID) (map[string]int64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"pollId": pollID}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$optionId",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	counts := map[string]int64{}
	var result struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	for cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		counts[result.ID] = result.Count
	}
	return counts, cursor.Err()
}

// CountByPoll returns the total number of votes cast in a poll.
func (r *VoteRepository) CountByPoll(ctx context.Context, pollID primitive.ObjectID) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"pollId": pollID})
}