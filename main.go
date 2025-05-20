package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/zulfkhar00/instafit_mvp/handlers"
	"github.com/zulfkhar00/instafit_mvp/internal/middleware"
	"github.com/zulfkhar00/instafit_mvp/services/storage"

	"github.com/joho/godotenv"
)

const (
	ServerPort = "8080"
)

func main() {
	// Default environment
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	envFile := fmt.Sprintf(".env.%s", env)
	// Load specific environment file, fallback to default .env
	if _, err := os.Stat(envFile); err == nil {
		log.Printf("Loading environment from %s", envFile)
	} else {
		log.Printf("%s not found, loading default .env file", envFile)
		envFile = ".env"
	}
	if err := godotenv.Load(envFile); err != nil {
		log.Fatalf("Error loading %s file: %v", envFile, err)
	}

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		log.Fatal("JWT_SECRET environment variable not set")
	}

	// Initialize services
	storageSvc, err := storage.NewR2Service()
	if err != nil {
		log.Fatalf("failed to initialize storage service: %v", err)
	}

	// initialize handlers
	clothesHandler := &handlers.ClothesHandler{
		Storage: storageSvc,
	}
	userHandler := &handlers.UserHandler{}

	// create a new Hertz server
	h := server.New(server.WithHostPorts(":" + ServerPort))

	// Set up routes
	h.GET("/api/health", func(ctx context.Context, c *app.RequestContext) {
		handlers.HealthCheckHandler(ctx, c)
	})
	// WARNING: This is a TESTING-ONLY route. Disable or remove in production!
	h.POST("/api/test-auth", userHandler.TestAuthHandler)

	authGroup := h.Group("/api")
	authGroup.Use(middleware.AuthMiddleware(jwtSecret))
	authGroup.POST("/wardrobe/add", clothesHandler.AddClothesToWardrobeHandler)
	authGroup.DELETE("/wardrobe/:clothId", clothesHandler.RemoveClothingFromWardrobeHandler)
	authGroup.POST("/virtual-tryon", func(ctx context.Context, c *app.RequestContext) {
		handlers.VirtualTryOnHandler(ctx, c)
	})

	// Start server
	log.Printf("Server starting on port %s...", ServerPort)
	h.Spin()
}
