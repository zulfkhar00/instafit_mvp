package handlers

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"

	"github.com/zulfkhar00/instafit_mvp/internal"
	"github.com/zulfkhar00/instafit_mvp/services/storage"

	"github.com/cloudwego/hertz/pkg/app"
)

type ClothesHandler struct {
	Storage storage.StorageService
}

// Handler for adding clothes to wardrobe endpoint
func (h *ClothesHandler) AddClothesToWardrobeHandler(ctx context.Context, c *app.RequestContext) {
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

	var wg sync.WaitGroup
	type UploadedResult struct {
		ImageURL   string                 `json:"image_url"`
		ClothingID string                 `json:"clothing_id"`
		Metadata   map[string]interface{} `json:"metadata"`
	}
	resultCh := make(chan UploadedResult, len(clothFiles)*5)
	errorCh := make(chan error, len(clothFiles))

	for fileIdx, file := range clothFiles {
		wg.Add(1)
		go func(imageFileIdx int, imageFile *multipart.FileHeader) {
			defer wg.Done()
			uploadedCloth, err := imageFile.Open()
			if err != nil {
				errorCh <- fmt.Errorf("failed to open file %d: %v", imageFileIdx, err)
				return
			}

			// Read file content into memory
			imgBytes, err := io.ReadAll(uploadedCloth)
			uploadedCloth.Close() // Always close after reading
			if err != nil {
				errorCh <- fmt.Errorf("failed to read file %d: %v", imageFileIdx, err)
				return
			}

			// segment image, retrieve metadata
			segmentedImages, err := internal.Segment_clothes(imgBytes)
			if err != nil {
				fmt.Printf("Segment_clothes error for file %d: %v", imageFileIdx, err)
				errorCh <- fmt.Errorf("segmentation failed for file %d: %v", imageFileIdx, err)
				return
			}
			if len(segmentedImages) == 0 {
				errorCh <- fmt.Errorf("no segmented images returned for file %d", imageFileIdx)
				return
			}

			for i, segmentedImg := range segmentedImages {
				filename := fmt.Sprintf("wardrobe/%s/%s.jpg", userId, segmentedImg.ID)
				contentType := "image/jpeg"
				url, err := h.Storage.UploadBlob(ctx, segmentedImg.Image, filename, contentType)
				if err != nil {
					errorCh <- fmt.Errorf("failed to upload segmented image %d_%d: %v", imageFileIdx, i, err)
					return
				}

				resultCh <- UploadedResult{
					ImageURL:   url,
					ClothingID: segmentedImg.ID,
					Metadata:   segmentedImg.Metadata,
				}
			}
		}(fileIdx, file)
	}

	// Close channels after all goroutines complete
	go func() {
		wg.Wait()
		close(resultCh)
		close(errorCh)
	}()

	// Collect uploaded results
	var uploadedItems []UploadedResult
	for result := range resultCh {
		uploadedItems = append(uploadedItems, result)
	}

	// Collect errors explicitly
	var firstErr error
	for err := range errorCh {
		if err != nil && firstErr == nil {
			firstErr = err // Take only the first error for simplicity
		}
	}
	// Check if any error occurred
	if firstErr != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   firstErr.Error(),
		})
		return
	}

	// TODO: insert `uploadedItems` into DB

	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"clothes": uploadedItems,
	})

}
