package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/zulfkhar00/instafit_mvp/internal"

	"github.com/cloudwego/hertz/pkg/app"
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

	var wg sync.WaitGroup
	resultCh := make(chan int, len(clothFiles))
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
			fmt.Printf("Segmented %d images for file %d", len(segmentedImages), imageFileIdx)
			if len(segmentedImages) == 0 {
				errorCh <- fmt.Errorf("no segmented images returned for file %d", imageFileIdx)
				return
			}

			for i, segmentedImg := range segmentedImages {
				imagePath := filepath.Join(TempDir, fmt.Sprintf("file_%d_%d.jpg", imageFileIdx, i))
				metadataPath := filepath.Join(TempDir, fmt.Sprintf("file_%d_%d.json", imageFileIdx, i))
				if err := os.WriteFile(imagePath, segmentedImg.Image, 0644); err != nil {
					errorCh <- fmt.Errorf("failed to write image file %d_%d: %v", imageFileIdx, i, err)
					return
				}
				// Convert metadata to JSON
				metadataJSON, err := json.MarshalIndent(segmentedImg.Metadata, "", "  ")
				if err != nil {
					errorCh <- fmt.Errorf("failed to marshal metadata %d_%d: %v", imageFileIdx, i, err)
					return
				}
				// Write metadata file
				if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
					errorCh <- fmt.Errorf("failed to write metadata file %d_%d: %v", imageFileIdx, i, err)
					return
				}
			}
			resultCh <- len(segmentedImages)
		}(fileIdx, file)
	}

	// Close channels after all goroutines complete
	go func() {
		wg.Wait()
		close(resultCh)
		close(errorCh)
	}()

	// Collect results and check for errors
	totalProcessed := 0
	for count := range resultCh {
		totalProcessed += count
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

	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"count":   totalProcessed,
	})

}
