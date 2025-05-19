package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/zulfkhar00/instafit_mvp/internal"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// Handler for adding clothes to wardrobe endpoint
func AddClothesToWardrobeHandler(ctx context.Context, c *app.RequestContext) {
	// Get files from form data
	form, err := c.MultipartForm()
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Failed to parse form: %v", err))
		return
	}

	clothFiles := form.File["clothes"]
	if len(clothFiles) == 0 {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "No cloth images uploaded",
		})
		return
	}

	processed := 0
	for _, file := range clothFiles {
		imageID := uuid.New().String()
		clothPath := filepath.Join(TempDir, fmt.Sprintf("cloth_%s%s", imageID, filepath.Ext(file.Filename)))

		if err := c.SaveUploadedFile(file, clothPath); err != nil {
			continue // Skip this file if there's an error
		}

		// segment image, retrieve metadata
		segmentedImages := internal.Segment_clothes(clothPath)
		for i, segmentedImg := range segmentedImages {
			imagePath := filepath.Join(TempDir, fmt.Sprintf("file_%d_%d.jpg", processed, i))
			metadataPath := filepath.Join(TempDir, fmt.Sprintf("file_%d_%d.json", processed, i))
			if err := os.WriteFile(imagePath, segmentedImg.Image, 0644); err != nil {
				c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to write image file: %v", err))
				return
			}
			// Convert metadata to JSON
			metadataJSON, err := json.MarshalIndent(segmentedImg.Metadata, "", "  ")
			if err != nil {
				c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal metadata: %v", err))
				return
			}
			// Write metadata file
			if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
				c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to write metadata file: %v", err))
				return
			}
		}
		processed++

		// Clean up temporary file
		defer os.Remove(clothPath)
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"count":   processed,
	})

}
