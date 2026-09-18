package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"poll-live/backend/config"
	"poll-live/backend/handlers"
	"poll-live/backend/repositories"
	"poll-live/backend/repositories/database"
	"poll-live/backend/routes"
	"poll-live/backend/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- MongoDB ---
	mongoClient, err := database.Connect(ctx, cfg.MongoURI)
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = database.Disconnect(shutdownCtx, mongoClient)
	}()

	db := mongoClient.Database(cfg.MongoDatabase)

	if err := database.EnsureIndexes(ctx, db); err != nil {
		log.Fatalf("mongodb indexes: %v", err)
	}

	// --- Redis ---
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)

	redisSvc := services.NewRedisService(redisClient)
	if err := redisSvc.Ping(ctx); err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisClient.Close()

	// --- Services ---
	userRepo := repositories.NewUserRepository(db)
	pollRepo := repositories.NewPollRepository(db)
	voteRepo := repositories.NewVoteRepository(db)

	authSvc := services.NewAuthService(userRepo, cfg)
	pollSvc := services.NewPollService(pollRepo, voteRepo, redisSvc)
	voteSvc := services.NewVoteService(pollRepo, voteRepo, redisSvc)
	wsHub := services.NewWebSocketHub(redisSvc)

	// --- Handlers ---
	allowedOrigins := []string{cfg.FrontendURL, "http://localhost:5173", "http://127.0.0.1:5173"}
	authHandler := handlers.NewAuthHandler(authSvc)
	pollHandler := handlers.NewPollHandler(pollSvc)
	voteHandler := handlers.NewVoteHandler(voteSvc)
	wsHandler := handlers.NewWebSocketHandler(wsHub, allowedOrigins)

	// --- Router ---
	router := routes.Setup(routes.Options{
		JWTSecret:        cfg.JWTSecret,
		FrontendURL:      cfg.FrontendURL,
		AuthHandler:      authHandler,
		PollHandler:      pollHandler,
		VoteHandler:      voteHandler,
		WebSocketHandler: wsHandler,
	})

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("PollLive backend listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}