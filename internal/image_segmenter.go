package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/google/uuid"
)

type SegmentedImage struct {
	Image    []byte                 `json:"image"`
	ID       string                 `json:"id"`
	Metadata map[string]interface{} `json:"metadata"`
}

func Segment_clothes(inputImage []byte) ([]SegmentedImage, error) {

	url := "http://127.0.0.1:8000/segment/"
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	// Attach the image file
	part, err := writer.CreateFormFile("file", "input.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}
	_, err = io.Copy(part, bytes.NewReader(inputImage))
	if err != nil {
		return nil, fmt.Errorf("failed to copy file: %v", err)
	}
	writer.Close()
	// Create HTTP request
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response from server: %s", string(respBody))
	}
	// Parse JSON response
	var parsedResponse struct {
		SegmentedImages []struct {
			Filename string                 `json:"filename"`
			Image    string                 `json:"image"`
			Metadata map[string]interface{} `json:"metadata"`
		} `json:"segmented_images"`
	}
	err = json.Unmarshal(respBody, &parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	// Decode images from base64 to bytes
	results := make([]SegmentedImage, 0, len(parsedResponse.SegmentedImages))
	for _, item := range parsedResponse.SegmentedImages {
		imgBytes, err := base64.StdEncoding.DecodeString(item.Image)
		if err != nil {
			fmt.Printf("base64 decode error: %v", err)
			return nil, fmt.Errorf("base64 decode error: %v", err)
		}
		results = append(results, SegmentedImage{
			Image:    imgBytes,
			ID:       uuid.NewString(),
			Metadata: item.Metadata,
		})
	}

	return results, nil
}
