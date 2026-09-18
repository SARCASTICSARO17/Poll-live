package services

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"poll-live/backend/models"
	"poll-live/backend/repositories"
)

// Vote-related domain errors.
var (
	ErrPollClosed  = errors.New("this poll is closed")
	ErrPollExpired = errors.New("this poll has expired")
	ErrVoteInvalid = errors.New("invalid vote")
)

// VoteService handles the cast-vote flow across MongoDB and Redis.
type VoteService struct {
	polls *repositories.PollRepository
	votes *repositories.VoteRepository
	redis *RedisService
}

// NewVoteService wires a vote service.
func NewVoteService(polls *repositories.PollRepository, votes *repositories.VoteRepository, redis *RedisService) *VoteService {
	return &VoteService{polls: polls, votes: votes, redis: redis}
}

// Vote validates and records a single vote, then increments Redis counters and
// publishes the updated results to the poll's Pub/Sub channel.
//
// MongoDB is written first: its unique index on (pollId, voterKey) atomically
// rejects duplicates, so a rejected duplicate never skews the Redis counter.
func (s *VoteService) Vote(ctx context.Context, pollID primitive.ObjectID, optionID string, voterKey string) error {
	poll, err := s.polls.FindByID(ctx, pollID)
	if err != nil {
		return err
	}

	if poll.Status != models.PollStatusActive {
		return ErrPollClosed
	}
	if poll.ExpiresAt != nil && time.Now().UTC().After(*poll.ExpiresAt) {
		return ErrPollExpired
	}

	validOption := false
	for _, opt := range poll.Options {
		if opt.ID == optionID {
			validOption = true
			break
		}
	}
	if !validOption {
		return ErrVoteInvalid
	}

	// Ensure Redis counters exist before voting so INCR always has a base.
	if err := s.redis.EnsureResults(ctx, poll.ID.Hex(), poll.Options, nil); err != nil {
		return err
	}

	vote := &models.Vote{
		PollID:   poll.ID,
		OptionID: optionID,
		VoterKey: voterKey,
	}
	if err := s.votes.Create(ctx, vote); err != nil {
		if errors.Is(err, repositories.ErrVoteAlreadyExists) {
			return err // surfaced verbatim to the client
		}
		return err
	}

	if err := s.redis.IncrementVote(ctx, poll.ID.Hex(), optionID); err != nil {
		return err
	}

	results, total, err := s.redis.GetResults(ctx, poll.ID.Hex(), poll.Options)
	if err != nil {
		return err
	}

	return s.redis.PublishUpdate(ctx, poll.ID.Hex(), PollUpdateEvent{
		Type:       "poll_update",
		PollID:     poll.ID.Hex(),
		TotalVotes: total,
		Results:    results,
	})
}