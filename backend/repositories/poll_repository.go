package repositories

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"poll-live/backend/models"
)

var ErrPollNotFound = errors.New("poll not found")

// PollRepository provides data access for polls.
type PollRepository struct {
	collection *mongo.Collection
}

// NewPollRepository creates a poll repository backed by MongoDB.
func NewPollRepository(db *mongo.Database) *PollRepository {
	return &PollRepository{collection: db.Collection("polls")}
}

// Create inserts a new poll and returns the stored poll.
func (r *PollRepository) Create(ctx context.Context, poll *models.Poll) (*models.Poll, error) {
	poll.ID = primitive.NewObjectID()
	poll.Status = models.PollStatusActive
	poll.CreatedAt = now()
	poll.UpdatedAt = now()
	_, err := r.collection.InsertOne(ctx, poll)
	if err != nil {
		return nil, err
	}
	return poll, nil
}

// FindByID finds an active (non-deleted) poll by its ObjectID.
func (r *PollRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Poll, error) {
	var poll models.Poll
	err := r.collection.FindOne(ctx, bson.M{"_id": id, "status": bson.M{"$ne": models.PollStatusDeleted}}).Decode(&poll)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrPollNotFound
	}
	if err != nil {
		return nil, err
	}
	return &poll, nil
}

// FindByCreator returns all non-deleted polls owned by a user, newest first.
func (r *PollRepository) FindByCreator(ctx context.Context, creatorID primitive.ObjectID) ([]models.Poll, error) {
	cursor, err := r.collection.Find(ctx,
		bson.M{"createdBy": creatorID, "status": bson.M{"$ne": models.PollStatusDeleted}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var polls []models.Poll
	if err := cursor.All(ctx, &polls); err != nil {
		return nil, err
	}
	return polls, nil
}

// UpdateStatus sets the status of a poll and refreshes UpdatedAt.
func (r *PollRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status string) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"status": status, "updatedAt": now()}},
	)
	return err
}

// Delete soft-deletes a poll by setting its status to deleted.
func (r *PollRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	return r.UpdateStatus(ctx, id, models.PollStatusDeleted)
}

func now() time.Time {
	return time.Now().UTC()
}