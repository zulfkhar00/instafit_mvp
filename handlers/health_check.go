package handlers

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/zulfkhar00/instafit_mvp/internal"
)

// Health check handler
func HealthCheckHandler(ctx context.Context, c *app.RequestContext) {
	comfyUIStatus := "not running"
	if internal.IsComfyUIRunning() {
		comfyUIStatus = "running"
	}

	response := map[string]string{
		"status":         "ok",
		"comfyui_status": comfyUIStatus,
	}

	c.JSON(200, response)
}
