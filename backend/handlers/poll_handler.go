package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"poll-live/backend/middleware"
	"poll-live/backend/models"
	"poll-live/backend/repositories"
	"poll-live/backend/services"
	"poll-live/backend/utils"
)

// PollHandler exposes poll lifecycle endpoints.
type PollHandler struct {
	polls *services.PollService
}

// NewPollHandler wires the poll handler.
func NewPollHandler(polls *services.PollService) *PollHandler {
	return &PollHandler{polls: polls}
}

type createPollRequest struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

type publicOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// publicPoll is a safe representation of a poll that never leaks internal or
// sensitive fields.
type publicPoll struct {
	ID        string          `json:"id"`
	Question  string          `json:"question"`
	Options   []publicOption  `json:"options"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
	ExpiresAt *time.Time      `json:"expiresAt,omitempty"`
}

func toPublicPoll(p *models.Poll) publicPoll {
	opts := make([]publicOption, len(p.Options))
	for i, o := range p.Options {
		opts[i] = publicOption{ID: o.ID, Text: o.Text}
	}
	return publicPoll{
		ID:        p.ID.Hex(),
		Question:  p.Question,
		Options:   opts,
		Status:    p.Status,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		ExpiresAt: p.ExpiresAt,
	}
}

// CreatePoll handles POST /api/polls (auth required).
func (h *PollHandler) CreatePoll(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserID)

	var req createPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	poll, err := h.polls.CreatePoll(c.Request.Context(), userID.(primitive.ObjectID), req.Question, req.Options)
	if err != nil {
		switch {
		case errors.Is(err, utils.ErrInvalidQuestion),
			errors.Is(err, utils.ErrInvalidOptions):
			utils.Error(c, http.StatusBadRequest, err.Error())
		default:
			utils.InternalError(c, err)
		}
		return
	}

	utils.Success(c, http.StatusCreated, gin.H{
		"poll": toPublicPoll(poll),
	})
}

// GetMyPolls handles GET /api/polls/my (auth required).
func (h *PollHandler) GetMyPolls(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserID)

	polls, totals, err := h.polls.GetMyPolls(c.Request.Context(), userID.(primitive.ObjectID))
	if err != nil {
		utils.InternalError(c, err)
		return
	}

	items := make([]gin.H, 0, len(polls))
	for i := range polls {
		p := &polls[i]
		items = append(items, gin.H{
			"poll":       toPublicPoll(p),
			"totalVotes": totals[p.ID.Hex()],
		})
	}

	utils.Success(c, http.StatusOK, gin.H{"polls": items})
}

// GetPoll handles GET /api/polls/:id (public).
func (h *PollHandler) GetPoll(c *gin.Context) {
	pollID, err := parsePollID(c)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid poll id")
		return
	}

	poll, results, total, err := h.polls.GetPoll(c.Request.Context(), pollID)
	if err != nil {
		if errors.Is(err, repositories.ErrPollNotFound) {
			utils.Error(c, http.StatusNotFound, "poll not found")
			return
		}
		utils.InternalError(c, err)
		return
	}

	isCreator, hasVoted := h.viewerState(c, poll)

	utils.Success(c, http.StatusOK, gin.H{
		"poll":       toPublicPoll(poll),
		"results":    results,
		"totalVotes": total,
		"isCreator":  isCreator,
		"hasVoted":   hasVoted,
	})
}

// GetResults handles GET /api/polls/:id/results (public).
func (h *PollHandler) GetResults(c *gin.Context) {
	pollID, err := parsePollID(c)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid poll id")
		return
	}

	poll, results, total, err := h.polls.GetPoll(c.Request.Context(), pollID)
	if err != nil {
		if errors.Is(err, repositories.ErrPollNotFound) {
			utils.Error(c, http.StatusNotFound, "poll not found")
			return
		}
		utils.InternalError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"pollId":     poll.ID.Hex(),
		"results":    results,
		"totalVotes": total,
	})
}

// ClosePoll handles PATCH /api/polls/:id/status (creator only).
func (h *PollHandler) ClosePoll(c *gin.Context) {
	pollID, err := parsePollID(c)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid poll id")
		return
	}
	userID, _ := c.Get(middleware.ContextUserID)

	if err := h.polls.ClosePoll(c.Request.Context(), pollID, userID.(primitive.ObjectID)); err != nil {
		if errors.Is(err, repositories.ErrPollNotFound) {
			utils.Error(c, http.StatusNotFound, "poll not found")
			return
		}
		if errors.Is(err, services.ErrForbidden) {
			utils.Error(c, http.StatusForbidden, err.Error())
			return
		}
		utils.InternalError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"message": "poll closed"})
}

// DeletePoll handles DELETE /api/polls/:id (creator only).
func (h *PollHandler) DeletePoll(c *gin.Context) {
	pollID, err := parsePollID(c)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid poll id")
		return
	}
	userID, _ := c.Get(middleware.ContextUserID)

	if err := h.polls.DeletePoll(c.Request.Context(), pollID, userID.(primitive.ObjectID)); err != nil {
		if errors.Is(err, repositories.ErrPollNotFound) {
			utils.Error(c, http.StatusNotFound, "poll not found")
			return
		}
		if errors.Is(err, services.ErrForbidden) {
			utils.Error(c, http.StatusForbidden, err.Error())
			return
		}
		utils.InternalError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"message": "poll deleted"})
}

// viewerState computes whether the current viewer is the poll creator and
// whether they have already voted in the poll.
func (h *PollHandler) viewerState(c *gin.Context, poll *models.Poll) (isCreator, hasVoted bool) {
	if authenticated, ok := c.Get(middleware.ContextAuthFlag); ok {
		if authed, _ := authenticated.(bool); authed {
			userID, _ := c.Get(middleware.ContextUserID)
			uid, _ := userID.(primitive.ObjectID)
			isCreator = poll.CreatedBy == uid
			voterKey := models.NewVoterKey(uid.Hex(), true)
			hasVoted, _ = h.polls.HasVoted(c.Request.Context(), poll.ID, models.VoterKey(voterKey))
			return isCreator, hasVoted
		}
	}

	if vid := c.GetHeader("X-Voter-Id"); vid != "" {
		hasVoted, _ = h.polls.HasVoted(c.Request.Context(), poll.ID, models.VoterKey(models.NewVoterKey(vid, false)))
	}
	return isCreator, hasVoted
}

func parsePollID(c *gin.Context) (primitive.ObjectID, error) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		return primitive.NilObjectID, err
	}
	return id, nil
}