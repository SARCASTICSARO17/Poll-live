package services

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"poll-live/backend/config"
	"poll-live/backend/models"
	"poll-live/backend/repositories"
	"poll-live/backend/utils"
)

// TokenTTL is how long JWTs remain valid.
const TokenTTL = 24 * time.Hour

// ErrInvalidCredentials is returned when the email/password pair is wrong.
var ErrInvalidCredentials = errors.New("invalid email or password")

// AuthService implements signup and login business logic.
type AuthService struct {
	users   *repositories.UserRepository
	config  *config.Config
}

// NewAuthService wires an auth service.
func NewAuthService(users *repositories.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{users: users, config: cfg}
}

// Signup validates credentials, hashes the password and creates a user.
func (s *AuthService) Signup(ctx context.Context, email, password string) (*models.User, error) {
	email, err := utils.ValidateEmail(email)
	if err != nil {
		return nil, err
	}
	if _, err := utils.ValidatePassword(password); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        email,
		PasswordHash: string(hash),
	}

	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repositories.ErrDuplicateEmail) {
			return nil, errors.New("an account with this email already exists")
		}
		return nil, err
	}
	return user, nil
}

// Login verifies credentials and returns a signed JWT plus the user.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, *models.User, error) {
	email, err := utils.ValidateEmail(email)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := utils.GenerateToken(user.ID, s.config.JWTSecret, TokenTTL)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}