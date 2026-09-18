package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"poll-live/backend/middleware"
	"poll-live/backend/models"
	"poll-live/backend/repositories"
	"poll-live/backend/services"
	"poll-live/backend/utils"
)

// VoteHandler exposes the vote endpoint.
type VoteHandler struct {
	votes *services.VoteService
}

// NewVoteHandler wires the vote handler.
func NewVoteHandler(votes *services.VoteService) *VoteHandler {
	return &VoteHandler{votes: votes}
}

type voteRequest struct {
	OptionID string `json:"optionId"`
}

// Vote handles POST /api/polls/:id/vote. It accepts authenticated and
// anonymous voters (anonymous voters must supply an X-Voter-Id header).
func (h *VoteHandler) Vote(c *gin.Context) {
	pollID, err := parsePollID(c)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid poll id")
		return
	}

	var req voteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	voterKey, err := h.voterKey(c)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.votes.Vote(c.Request.Context(), pollID, req.OptionID, voterKey); err != nil {
		switch {
		case errors.Is(err, repositories.ErrPollNotFound):
			utils.Error(c, http.StatusNotFound, "poll not found")
		case errors.Is(err, repositories.ErrVoteAlreadyExists):
			utils.Error(c, http.StatusConflict, "You have already voted in this poll.")
		case errors.Is(err, services.ErrPollClosed),
			errors.Is(err, services.ErrPollExpired),
			errors.Is(err, services.ErrVoteInvalid):
			utils.Error(c, http.StatusBadRequest, err.Error())
		default:
			utils.InternalError(c, err)
		}
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"message": "vote recorded"})
}

// voterKey resolves the stable identity used for duplicate-vote prevention.
func (h *VoteHandler) voterKey(c *gin.Context) (string, error) {
	if authenticated, ok := c.Get(middleware.ContextAuthFlag); ok {
		if authed, _ := authenticated.(bool); authed {
			userID, _ := c.Get(middleware.ContextUserID)
			uid, _ := userID.(primitive.ObjectID)
			return models.NewVoterKey(uid.Hex(), true), nil
		}
	}

	vid := c.GetHeader("X-Voter-Id")
	if vid == "" {
		return "", errors.New("voter identifier required")
	}
	return models.NewVoterKey(vid, false), nil
}