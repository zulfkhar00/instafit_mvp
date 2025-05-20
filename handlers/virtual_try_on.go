package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/zulfkhar00/instafit_mvp/internal"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

const (
	ComfyUIAPIURL = "http://127.0.0.1:8188/api"
	WorkflowPath  = "./ImageWorkflow.json"
)

var (
	TempDir = getTempDir()
)

// Handler for virtual try-on endpoint
func VirtualTryOnHandler(ctx context.Context, c *app.RequestContext) {
	// Ensure ComfyUI is running
	if !internal.IsComfyUIRunning() {
		if err := internal.StartComfyUI(); err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to start ComfyUI: %v", err))
			return
		}

		// Double-check if it's running
		if !internal.IsComfyUIRunning() {
			c.String(http.StatusInternalServerError, "ComfyUI failed to start")
			return
		}
	}

	// Get files from form data
	form, err := c.MultipartForm()
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Failed to parse form: %v", err))
		return
	}

	// Get input files
	personFiles := form.File["person_image"]
	if len(personFiles) == 0 {
		c.String(http.StatusBadRequest, "person_image is required")
		return
	}
	personHeader := personFiles[0]

	garmentFiles := form.File["garment_image"]
	if len(garmentFiles) == 0 {
		c.String(http.StatusBadRequest, "garment_image is required")
		return
	}
	garmentHeader := garmentFiles[0]

	// Get prompt parameter or use default
	prompt := c.PostForm("prompt")
	if prompt == "" {
		prompt = "shirt"
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Save uploaded files temporarily
	personPath := filepath.Join(TempDir, fmt.Sprintf("person_%s%s", sessionID, filepath.Ext(personHeader.Filename)))
	if err := c.SaveUploadedFile(personHeader, personPath); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to save person image: %v", err))
		return
	}
	defer os.Remove(personPath)

	garmentPath := filepath.Join(TempDir, fmt.Sprintf("garment_%s%s", sessionID, filepath.Ext(garmentHeader.Filename)))
	if err := c.SaveUploadedFile(garmentHeader, garmentPath); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to save garment image: %v", err))
		return
	}
	defer os.Remove(garmentPath)

	log.Printf("Images saved: %s, %s", personPath, garmentPath)

	// RUN COMFYUI WORKFLOW
	fileData, err := os.ReadFile(WorkflowPath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
	// Parse the JSON
	var workflow map[string]interface{}
	err = json.Unmarshal(fileData, &workflow)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}
	// set the prompt of masker
	if node21, ok := workflow["21"].(map[string]interface{}); ok {
		if inputs, ok := node21["inputs"].(map[string]interface{}); ok {
			inputs["prompt"] = prompt // Change the prompt dynamically
		}
	}
	// set the image path of garment image
	if node22, ok := workflow["22"].(map[string]interface{}); ok {
		if inputs, ok := node22["inputs"].(map[string]interface{}); ok {
			inputs["image"] = garmentPath
		}
	}
	// set the image path of person image
	if node27, ok := workflow["27"].(map[string]interface{}); ok {
		if inputs, ok := node27["inputs"].(map[string]interface{}); ok {
			inputs["image"] = personPath
		}
	}

	images, err := internal.GetImages(workflow)
	if err != nil {
		log.Fatal(err)
	}
	var imageData []byte
	found := false
	for _, imageList := range images {
		if len(imageList) > 0 {
			imageData = imageList[0]
			found = true
			break
		}
	}
	if !found {
		log.Fatal("No image received from ComfyUI")
	}

	// Set appropriate headers
	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=vton_result_%s.jpg", sessionID))

	// Send the image in response
	c.Data(http.StatusOK, "image/jpeg", imageData)

	// Clean up temp files
	os.Remove(personPath)
	os.Remove(garmentPath)
}

func getTempDir() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(wd, "temp_files")
}
