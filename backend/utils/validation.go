package utils

import (
	"errors"
	"regexp"
	"strings"
)

// Validation errors shared across services.
var (
	ErrInvalidEmail    = errors.New("a valid email is required")
	ErrInvalidPassword = errors.New("password must be at least 6 characters")
	ErrInvalidQuestion = errors.New("question is required and must be at most 200 characters")
	ErrInvalidOptions  = errors.New("poll must have between 2 and 6 non-empty, unique options")
	ErrInvalidOptionID = errors.New("invalid option id")
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

const (
	MaxQuestionLen = 200
	MaxOptionLen   = 100
	MinOptions     = 2
	MaxOptions     = 6
	MinPasswordLen = 6
)

// ValidateEmail normalises and validates an email address.
func ValidateEmail(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || !emailRegex.MatchString(email) || len(email) > 254 {
		return "", ErrInvalidEmail
	}
	return email, nil
}

// ValidatePassword checks password strength and returns it unchanged.
func ValidatePassword(password string) (string, error) {
	if len(password) < MinPasswordLen {
		return "", ErrInvalidPassword
	}
	return password, nil
}

// ValidateQuestion trims and validates a poll question.
func ValidateQuestion(question string) (string, error) {
	question = strings.TrimSpace(question)
	if question == "" || len(question) > MaxQuestionLen {
		return "", ErrInvalidQuestion
	}
	return question, nil
}

// ValidateOptions trims, deduplicates (case-insensitive) and validates a poll's
// options slice. It returns the cleaned options slice.
func ValidateOptions(options []string) ([]string, error) {
	if len(options) < MinOptions || len(options) > MaxOptions {
		return nil, ErrInvalidOptions
	}

	cleaned := make([]string, 0, len(options))
	seen := make(map[string]struct{}, len(options))
	for _, opt := range options {
		opt = strings.TrimSpace(opt)
		if opt == "" || len(opt) > MaxOptionLen {
			return nil, ErrInvalidOptions
		}
		key := strings.ToLower(opt)
		if _, exists := seen[key]; exists {
			return nil, ErrInvalidOptions
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, opt)
	}
	return cleaned, nil
}