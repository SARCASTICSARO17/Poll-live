package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"poll-live/backend/services"
	"poll-live/backend/utils"
)

// AuthHandler exposes authentication endpoints.
type AuthHandler struct {
	auth *services.AuthService
}

// NewAuthHandler wires the auth handler.
func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Signup handles POST /api/auth/signup.
func (h *AuthHandler) Signup(c *gin.Context) {
	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.auth.Signup(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, utils.ErrInvalidEmail),
			errors.Is(err, utils.ErrInvalidPassword):
			utils.Error(c, http.StatusBadRequest, err.Error())
		default:
			utils.Error(c, http.StatusConflict, err.Error())
		}
		return
	}

	utils.Success(c, http.StatusCreated, gin.H{
		"id":       user.ID.Hex(),
		"email":    user.Email,
		"message":  "account created successfully",
	})
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	token, user, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			utils.Error(c, http.StatusUnauthorized, "invalid email or password")
			return
		}
		utils.InternalError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    user.ID.Hex(),
			"email": user.Email,
		},
	})
}