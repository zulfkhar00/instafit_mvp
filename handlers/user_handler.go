package handlers

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/zulfkhar00/instafit_mvp/internal/auth"
)

type UserHandler struct{}

// TestAuthHandler generates JWT tokens for testing purposes only.
func (h *UserHandler) TestAuthHandler(ctx context.Context, c *app.RequestContext) {
	type AuthTestRequest struct {
		UserId string `json:"user_id" binding:"required"`
	}

	var req AuthTestRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "user_id is required",
		})
		return
	}

	token, err := auth.GenerateJWT(req.UserId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "Failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   token,
	})
}
