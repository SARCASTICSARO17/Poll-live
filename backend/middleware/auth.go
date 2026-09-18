package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"poll-live/backend/utils"
)

const (
	ContextUserID  = "userId"
	ContextAuthFlag = "authenticated"
)

// AuthRequired rejects requests without a valid Bearer JWT. It extracts and
// stores the user ID on the context for downstream handlers.
func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c)
		if !ok {
			utils.Error(c, http.StatusUnauthorized, "authentication required")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(token, jwtSecret)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		userID, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "invalid token payload")
			c.Abort()
			return
		}

		c.Set(ContextUserID, userID)
		c.Set(ContextAuthFlag, true)
		c.Next()
	}
}

// OptionalAuth parses the JWT when present but never blocks the request. It is
// used on public endpoints so handlers can personalise responses for logged-in
// users (for example marking whether they already voted).
func OptionalAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ContextAuthFlag, false)

		token, ok := bearerToken(c)
		if !ok {
			c.Next()
			return
		}

		claims, err := utils.ParseToken(token, jwtSecret)
		if err != nil {
			c.Next()
			return
		}

		userID, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			c.Next()
			return
		}

		c.Set(ContextUserID, userID)
		c.Set(ContextAuthFlag, true)
		c.Next()
	}
}

func bearerToken(c *gin.Context) (string, bool) {
	header := c.GetHeader("Authorization")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}