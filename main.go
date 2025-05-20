package main

import (
	"context"
	"log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/zulfkhar00/instafit_mvp/handlers"
)

const (
	ServerPort = "8080"
)

func main() {
	// create a new Hertz server
	h := server.New(server.WithHostPorts(":" + ServerPort))

	// Set up routes
	h.POST("/api/virtual-tryon", func(ctx context.Context, c *app.RequestContext) {
		handlers.VirtualTryOnHandler(ctx, c)
	})
	h.POST("/api/wardrobe/add", func(ctx context.Context, c *app.RequestContext) {
		handlers.AddClothesToWardrobeHandler(ctx, c)
	})
	h.GET("/api/health", func(ctx context.Context, c *app.RequestContext) {
		handlers.HealthCheckHandler(ctx, c)
	})

	// Start server
	log.Printf("Server starting on port %s...", ServerPort)
	h.Spin()
}
