package internal

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	// For joining paths safely
)

type SegmentedImage struct {
	Image    []byte                 `json:"image"`
	Metadata map[string]interface{} `json:"metadata"`
}

func Segment_clothes(inputImagePath string) []SegmentedImage {
	pythonScript := "/Users/zmaukey/Desktop/vton_mvp/internal/image_segmentator/process.py" // Path to your Python script
	outputDir := "/Users/zmaukey/Desktop/vton_mvp/output"                                   // Directory to save extracted images

	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory %s: %v\n", outputDir, err)
		os.Exit(1)
	}

	// Create a command to run the Python script
	cmd := exec.Command("python", pythonScript, "--image", inputImagePath)

	// Create a buffer to capture stdout (the zip data)
	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf

	// Capture stderr for Python errors and logs
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	fmt.Println("Running Python script...")

	// Run the command
	err := cmd.Run()

	// Always print stderr output for debugging
	if stderrBuf.Len() > 0 {
		fmt.Fprintf(os.Stderr, "--- Python stderr ---\n%s\n---------------------\n", stderrBuf.String())
	}

	// Check for command execution errors
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running python script: %v\n", err)
		os.Exit(1)
	}

	// Get the captured stdout bytes (should be the zip archive)
	zipData := stdoutBuf.Bytes()

	if len(zipData) == 0 {
		fmt.Println("Error: Python script produced no output (no zip data).")
		os.Exit(1)
	}

	fmt.Printf("Successfully captured %d bytes from Python (expected zip data).\n", len(zipData))

	// --- Read the ZIP archive from the captured bytes ---
	zipReader := bytes.NewReader(zipData)
	zipArchive, err := zip.NewReader(zipReader, int64(len(zipData)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading zip data: %v\n", err)
		os.Exit(1)
	}

	// --- Extract files from the ZIP archive ---
	fmt.Println("Extracting segmented clothing images:")
	extractedCount := 0

	filenameToMetadata := make(map[string]map[string]interface{})
	filenameToImg := make(map[string][]byte)
	for _, file := range zipArchive.File {
		// Prevent directory traversal attacks (though less likely with in-memory zip from trusted script)
		// and skip directories if any were somehow included.
		if file.Mode().IsDir() {
			continue
		}

		fmt.Printf("  - Extracting %s...\n", file.Name)

		// Open the file inside the zip archive
		zipFileReader, err := file.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file %s in zip: %v\n", file.Name, err)
			continue // Continue to the next file
		}
		// Ensure the reader is closed when we're done with this file
		defer zipFileReader.Close() // This defer is inside the loop, but it's okay as it's tied to the zipFileReader scope

		// Read the file content
		data, err := io.ReadAll(zipFileReader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading data from %s: %v\n", file.Name, err)
			continue
		}
		if file.Name[len(file.Name)-4:] == ".png" {
			filenameToImg[file.Name[:len(file.Name)-4]] = data
		} else {
			// Parse the metadata (assuming it's JSON)
			metadata := make(map[string]interface{})
			if err := json.Unmarshal(data, &metadata); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing metadata from %s: %v\n", file.Name, err)
				continue
			}
			filenameToMetadata[file.Name[:len(file.Name)-5]] = metadata
		}
		// // Determine the output path for the extracted file
		// extractedFilePath := filepath.Join(outputDir, file.Name)

		// // Save the extracted image data to a file
		// err = os.WriteFile(extractedFilePath, imageData, 0644)
		// if err != nil {
		// 	fmt.Fprintf(os.Stderr, "Error saving extracted file %s: %v\n", extractedFilePath, err)
		// 	continue
		// }
		// fmt.Printf("    Saved to %s\n", extractedFilePath)
		extractedCount++
	}

	result := make([]SegmentedImage, 0)
	for filename := range filenameToImg {
		img := filenameToImg[filename]
		metadata := filenameToMetadata[filename]
		result = append(result, SegmentedImage{Image: img, Metadata: metadata})
	}

	fmt.Printf("Extraction complete. Extracted %d image(s).\n", extractedCount)

	return result
}
