package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	ComfyUIPath   = "/Users/zmaukey/ComfyUI"
	ComfyUIAPIURL = "127.0.0.1:8188"
)

var clientID = uuid.NewString()

var (
	comfyUICmd *exec.Cmd
	cmdMutex   sync.Mutex
)

// Start ComfyUI as a background process
func StartComfyUI() error {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	env := os.Environ()
	env = append(env,
		"HTTP_PROXY=",
		"HTTPS_PROXY=",
		"http_proxy=",
		"https_proxy=",
		"NO_PROXY=*",
		"no_proxy=*")

	log.Println("Starting ComfyUI...")
	comfyUICmd = exec.Command("python", "main.py", "--listen", "127.0.0.1")
	comfyUICmd.Env = env
	comfyUICmd.Dir = ComfyUIPath
	comfyUICmd.Stdout = os.Stdout
	comfyUICmd.Stderr = os.Stderr

	if err := comfyUICmd.Start(); err != nil {
		return fmt.Errorf("failed to start ComfyUI: %v", err)
	}

	// Wait for ComfyUI to initialize
	time.Sleep(10 * time.Second)
	log.Println("ComfyUI started")
	return nil
}

// Check if ComfyUI is running
func IsComfyUIRunning() bool {
	resp, err := http.Get(ComfyUIAPIURL + "/system_stats")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func queuePrompt(prompt map[string]interface{}) (string, error) {
	payload := map[string]interface{}{
		"prompt":    prompt,
		"client_id": clientID,
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post("http://"+ComfyUIAPIURL+"/prompt", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result["prompt_id"].(string), nil
}

func GetImages(prompt map[string]interface{}) (map[string][][]byte, error) {

	// Marshal to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(prompt, "", "  ")
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		return nil, err
	}
	// Write to file
	err = os.WriteFile("generatede_workflow.json", jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return nil, err
	}

	promptID, err := queuePrompt(prompt)
	if err != nil {
		return nil, err
	}

	// Connect to WebSocket
	u := url.URL{Scheme: "ws", Host: ComfyUIAPIURL, Path: "/ws", RawQuery: "clientId=" + clientID}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	outputImages := make(map[string][][]byte)
	var currentNode string
	receivedMsgs := make([]map[string]interface{}, 0)

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.TextMessage {
			var message map[string]interface{}
			json.Unmarshal(msg, &message)

			receivedMsgs = append(receivedMsgs, message)

			// if message["type"] == "progress" {
			// 	continue
			// }

			// if message["type"] == "execution_success" {
			// 	continue
			// }

			if message["type"] == "executing" {
				data := message["data"].(map[string]interface{})
				if data["prompt_id"] == promptID {
					if data["node"] == nil {
						break // Done
					}
					currentNode = data["node"].(string)
				}
			}
		} else if msgType == websocket.BinaryMessage {
			receivedMsgs = append(receivedMsgs, map[string]interface{}{"img": msg})
			if currentNode == "30" {
				outputImages[currentNode] = append(outputImages[currentNode], msg[8:]) // Strip 8-byte prefix
			}
		}
	}

	// Marshal to JSON with indentation for readability
	jsonData, err = json.MarshalIndent(receivedMsgs, "", "  ")
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		return nil, err
	}
	// Write to file
	err = os.WriteFile("received_images.json", jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return nil, err
	}

	return outputImages, nil
}

func saveImage(imgBytes []byte, filename string) error {
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return err
	}
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}
