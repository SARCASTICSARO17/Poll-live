package services

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"poll-live/backend/models"
	"poll-live/backend/repositories"
	"poll-live/backend/utils"
)

// ErrForbidden is returned when a user attempts an action they are not allowed
// to perform (e.g. closing or deleting a poll they do not own).
var ErrForbidden = errors.New("you do not have permission to modify this poll")

// PollService contains the business logic for poll lifecycle. It composes the
// poll repository (MongoDB, permanent store) with the votes repository and the
// Redis service (live counters + Pub/Sub notifications).
type PollService struct {
	polls *repositories.PollRepository
	votes *repositories.VoteRepository
	redis *RedisService
}

// NewPollService wires a PollService.
func NewPollService(
	polls *repositories.PollRepository,
	votes *repositories.VoteRepository,
	redis *RedisService,
) *PollService {
	return &PollService{polls: polls, votes: votes, redis: redis}
}

// CreatePoll validates and stores a new poll, then seeds its live counters in
// Redis so that every option starts at zero.
func (s *PollService) CreatePoll(
	ctx context.Context,
	creatorID primitive.ObjectID,
	question string,
	options []string,
) (*models.Poll, error) {
	question, err := utils.ValidateQuestion(question)
	if err != nil {
		return nil, err
	}
	cleaned, err := utils.ValidateOptions(options)
	if err != nil {
		return nil, err
	}

	pollOptions := make([]models.Option, len(cleaned))
	for i, text := range cleaned {
		pollOptions[i] = models.Option{
			ID:   fmt.Sprintf("opt_%d", i+1),
			Text: text,
		}
	}

	poll := &models.Poll{
		Question:  question,
		Options:   pollOptions,
		CreatedBy: creatorID,
	}

	created, err := s.polls.Create(ctx, poll)
	if err != nil {
		return nil, err
	}

	// Seed live counters for every option so results are immediately available.
	if err := s.redis.EnsureResults(ctx, created.ID.Hex(), created.Options, nil); err != nil {
		return nil, err
	}
	return created, nil
}

// GetPoll returns a poll together with its live results and the total number
// of votes cast.
func (s *PollService) GetPoll(
	ctx context.Context,
	pollID primitive.ObjectID,
) (*models.Poll, []OptionResult, int64, error) {
	poll, err := s.polls.FindByID(ctx, pollID)
	if err != nil {
		return nil, nil, 0, err
	}

	results, total, err := s.liveResults(ctx, poll)
	if err != nil {
		return nil, nil, 0, err
	}
	return poll, results, total, nil
}

// GetMyPolls returns all polls owned by a user, together with a map from poll
// id to the live total number of votes for that poll.
func (s *PollService) GetMyPolls(
	ctx context.Context,
	creatorID primitive.ObjectID,
) ([]models.Poll, map[string]int64, error) {
	polls, err := s.polls.FindByCreator(ctx, creatorID)
	if err != nil {
		return nil, nil, err
	}

	totals := make(map[string]int64, len(polls))
	for i := range polls {
		p := &polls[i]
		_, total, err := s.liveResults(ctx, p)
		if err != nil {
			return nil, nil, err
		}
		totals[p.ID.Hex()] = total
	}
	return polls, totals, nil
}

// HasVoted reports whether a given voter has already cast a vote in a poll.
func (s *PollService) HasVoted(
	ctx context.Context,
	pollID primitive.ObjectID,
	voterKey models.VoterKey,
) (bool, error) {
	return s.votes.ExistsByVoterKey(ctx, pollID, string(voterKey))
}

// ClosePoll transitions a poll to the closed state. Only the creator may close
// a poll; the new status is broadcast over Redis Pub/Sub so connected clients
// update in real time.
func (s *PollService) ClosePoll(
	ctx context.Context,
	pollID primitive.ObjectID,
	userID primitive.ObjectID,
) error {
	poll, err := s.polls.FindByID(ctx, pollID)
	if err != nil {
		return err
	}
	if poll.CreatedBy != userID {
		return ErrForbidden
	}
	if err := s.polls.UpdateStatus(ctx, pollID, models.PollStatusClosed); err != nil {
		return err
	}

	if err := s.redis.PublishUpdate(ctx, pollID.Hex(), PollUpdateEvent{
		Type:   "poll_closed",
		PollID: pollID.Hex(),
	}); err != nil {
		return err
	}
	return nil
}

// DeletePoll removes a poll from the store. Only the creator may delete it.
func (s *PollService) DeletePoll(
	ctx context.Context,
	pollID primitive.ObjectID,
	userID primitive.ObjectID,
) error {
	poll, err := s.polls.FindByID(ctx, pollID)
	if err != nil {
		return err
	}
	if poll.CreatedBy != userID {
		return ErrForbidden
	}
	return s.polls.Delete(ctx, pollID)
}

// liveResults ensures the Redis counters for a poll exist and then reads the
// current results and total.
func (s *PollService) liveResults(
	ctx context.Context,
	poll *models.Poll,
) ([]OptionResult, int64, error) {
	if err := s.ensureSeeded(ctx, poll); err != nil {
		return nil, 0, err
	}
	return s.redis.GetResults(ctx, poll.ID.Hex(), poll.Options)
}

// ensureSeeded seeds Redis counters from MongoDB when they are missing. This is
// idempotent: it only inserts counters that do not already exist.
func (s *PollService) ensureSeeded(ctx context.Context, poll *models.Poll) error {
	seed, err := s.votes.CountByOption(ctx, poll.ID)
	if err != nil {
		return err
	}
	return s.redis.EnsureResults(ctx, poll.ID.Hex(), poll.Options, seed)
}
