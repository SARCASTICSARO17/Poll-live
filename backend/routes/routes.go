package routes

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"poll-live/backend/handlers"
	"poll-live/backend/middleware"
)

// Options bundles the dependencies needed to register all routes.
type Options struct {
	JWTSecret       string
	FrontendURL     string
	AuthHandler     *handlers.AuthHandler
	PollHandler     *handlers.PollHandler
	VoteHandler     *handlers.VoteHandler
	WebSocketHandler *handlers.WebSocketHandler
}

// Setup configures the Gin engine, middleware and all routes.
func Setup(opts Options) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{opts.FrontendURL, "http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Voter-Id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")

	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth
	auth := api.Group("/auth")
	{
		auth.POST("/signup", opts.AuthHandler.Signup)
		auth.POST("/login", opts.AuthHandler.Login)
	}

	// Polls
	polls := api.Group("/polls")
	polls.Use(middleware.OptionalAuth(opts.JWTSecret))
	{
		polls.GET("/my", middleware.AuthRequired(opts.JWTSecret), opts.PollHandler.GetMyPolls)
		polls.POST("", middleware.AuthRequired(opts.JWTSecret), opts.PollHandler.CreatePoll)
		polls.GET("/:id", opts.PollHandler.GetPoll)
		polls.GET("/:id/results", opts.PollHandler.GetResults)
		polls.GET("/:id/ws", opts.WebSocketHandler.Serve)
		polls.POST("/:id/vote", opts.VoteHandler.Vote)
		polls.PATCH("/:id/status", middleware.AuthRequired(opts.JWTSecret), opts.PollHandler.ClosePoll)
		polls.DELETE("/:id", middleware.AuthRequired(opts.JWTSecret), opts.PollHandler.DeletePoll)
	}

	return r
}