package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *ClothesHandler) RemoveClothingFromWardrobeHandler(ctx context.Context, c *app.RequestContext) {
	// Extract userId from context
	userIdVal, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "user ID missing from context",
		})
		return
	}
	userId, ok := userIdVal.(string)
	if !ok || userId == "" {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "invalid user ID format in context",
		})
		return
	}

	// Extract clothId from route parameters
	clothId := c.Param("clothId")
	if clothId == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "cloth ID required",
		})
		return
	}

	// Construct Cloudflare/S3 object key
	objectKey := fmt.Sprintf("wardrobe/%s/%s.jpg", userId, clothId)

	// Remove image from storage
	err := h.Storage.DeleteBlob(ctx, objectKey)
	if err != nil {
		log.Printf("Error deleting cloth %s for user %s: %v", clothId, userId, err)
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to delete clothing item: %v", err),
		})
		return
	}

	// TODO: Also remove the clothing entry from your database, if applicable.

	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"clothId": clothId,
	})
}
