package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"

	"poll-live/backend/models"
)

// OptionResult is the live result for a single poll option.
type OptionResult struct {
	OptionID   string  `json:"optionId"`
	Option     string  `json:"option"`
	Votes      int64   `json:"votes"`
	Percentage float64 `json:"percentage"`
}

// PollUpdateEvent is the JSON payload broadcast to WebSocket clients.
// It is also the payload published over Redis Pub/Sub.
type PollUpdateEvent struct {
	Type       string         `json:"type"`
	PollID     string         `json:"pollId"`
	TotalVotes int64          `json:"totalVotes"`
	Results    []OptionResult `json:"results"`
}

// RedisService owns all Redis interactions: live counters, atomic increments
// and Pub/Sub publication for realtime updates.
type RedisService struct {
	client *redis.Client

	ensureMu sync.Mutex
}

// NewRedisService creates a Redis wrapper.
func NewRedisService(client *redis.Client) *RedisService {
	return &RedisService{client: client}
}

// ResultKey returns the key under which a poll's live results are stored.
// Format: poll:{pollId}:results
func ResultKey(pollID string) string {
	return fmt.Sprintf("poll:%s:results", pollID)
}

// OptionCounterKey returns the per-option atomic counter key.
// Format: poll:{pollId}:option:{optionId}
func OptionCounterKey(pollID, optionID string) string {
	return fmt.Sprintf("poll:%s:option:%s", pollID, optionID)
}

// UpdateChannel returns the Pub/Sub channel for a poll's updates.
// Format: poll:{pollId}:updates
func UpdateChannel(pollID string) string {
	return fmt.Sprintf("poll:%s:updates", pollID)
}

// Ping verifies connectivity with Redis.
func (s *RedisService) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// EnsureResults initializes a poll's live counters in Redis from the provided
// seed counts (read from MongoDB) if they are not already present. It is safe
// to call concurrently: a single Redis Lua script guards against double
// initialization so live counters are never skewed.
func (s *RedisService) EnsureResults(ctx context.Context, pollID string, options []models.Option, seed map[string]int64) error {
	initScript := redis.NewScript(`
local exists = redis.call('EXISTS', KEYS[1])
if exists == 1 then
  return 1
end
for i = 1, #ARGV, 2 do
  redis.call('HSET', KEYS[1], ARGV[i], ARGV[i + 1])
end
return 0
`)

	args := make([]interface{}, 0, len(options)*2)
	for _, opt := range options {
		count := seed[opt.ID]
		args = append(args, opt.ID, count)
	}

	s.ensureMu.Lock()
	defer s.ensureMu.Unlock()

	_, err := initScript.Run(ctx, s.client, []string{ResultKey(pollID)}, args...).Result()
	return err
}

// IncrementVote atomically increments the live counter for an option and its
// entry in the poll's results hash. INCR guarantees correctness under
// concurrent votes; a single Lua script keeps both locations in sync.
func (s *RedisService) IncrementVote(ctx context.Context, pollID, optionID string) error {
	incrScript := redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
redis.call('HINCRBY', KEYS[2], ARGV[1], 1)
return n
`)

	_, err := incrScript.Run(ctx, s.client,
		[]string{OptionCounterKey(pollID, optionID), ResultKey(pollID)},
		optionID,
	).Int64()
	return err
}

// GetResults reads a poll's live results from Redis and enriches them with the
// option text and computed percentages. The returned order follows the poll's
// option order.
func (s *RedisService) GetResults(ctx context.Context, pollID string, options []models.Option) ([]OptionResult, int64, error) {
	values, err := s.client.HGetAll(ctx, ResultKey(pollID)).Result()
	if err != nil {
		return nil, 0, err
	}

	results := make([]OptionResult, 0, len(options))
	var totalVotes int64

	for _, opt := range options {
		votes, _ := parseInt64(values[opt.ID])
		totalVotes += votes
		results = append(results, OptionResult{
			OptionID: opt.ID,
			Option:   opt.Text,
			Votes:    votes,
		})
	}

	for i := range results {
		results[i].Percentage = percentage(results[i].Votes, totalVotes)
	}

	return results, totalVotes, nil
}

// PublishUpdate publishes a PollUpdateEvent to a poll's Pub/Sub channel so all
// running backend instances can rebroadcast it to their connected clients.
func (s *RedisService) PublishUpdate(ctx context.Context, pollID string, event PollUpdateEvent) error {
	data, err := jsonMarshal(event)
	if err != nil {
		return err
	}
	return s.client.Publish(ctx, UpdateChannel(pollID), data).Err()
}

// PublishStatusChange publishes a poll status event (e.g. closed) so listening
// clients can react immediately.
func (s *RedisService) PublishStatusChange(ctx context.Context, pollID, status string) error {
	event := map[string]string{
		"type":   "poll_status",
		"pollId": pollID,
		"status": status,
	}
	data, err := jsonMarshal(event)
	if err != nil {
		return err
	}
	return s.client.Publish(ctx, UpdateChannel(pollID), data).Err()
}

// Subscribe opens a Pub/Sub subscription for a channel. The caller owns the
// subscription and must Close it when no longer needed.
func (s *RedisService) Subscribe(ctx context.Context, channel string) (*redis.PubSub, error) {
	sub := s.client.Subscribe(ctx, channel)
	_, err := sub.Receive(ctx)
	if err != nil {
		sub.Close()
		return nil, err
	}
	return sub, nil
}

// GetClient exposes the underlying Redis client (for tests).
func (s *RedisService) GetClient() *redis.Client {
	return s.client
}

func percentage(votes, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(votes) / float64(total) * 100
}