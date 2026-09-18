package repositories

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"poll-live/backend/models"
)

var ErrUserNotFound = errors.New("user not found")
var ErrDuplicateEmail = errors.New("email already registered")

// UserRepository provides data access for users.
type UserRepository struct {
	collection *mongo.Collection
}

// NewUserRepository creates a user repository backed by MongoDB.
func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{collection: db.Collection("users")}
}

// Create inserts a new user. It returns ErrDuplicateEmail on unique conflicts.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.ID = primitive.NewObjectID()
	user.CreatedAt = now()
	_, err := r.collection.InsertOne(ctx, user)

	var mongoErr mongo.WriteException
	if errors.As(err, &mongoErr) {
		for _, we := range mongoErr.WriteErrors {
			if we.Code == 11000 {
				return ErrDuplicateEmail
			}
		}
	}
	return err
}

// FindByEmail finds a user by email.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID finds a user by its ObjectID.
func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}