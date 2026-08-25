package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Seedance 2.5 Implementation Example
// Add this to your leo2api or use as standalone reference

const (
	GraphQLURL = "https://api.leonardo.ai/v1/graphql"
)

// GraphQL request structure
type GraphQLRequest struct {
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
	Query         string                 `json:"query"`
}

// Generation request
type Seedance25Request struct {
	Model  string                 `json:"model"`
	Public bool                   `json:"public"`
	Params map[string]interface{} `json:"parameters"`
}

// Generate mutation
const generateMutation = `mutation Generate($request: CreateGenerationRequest!) {
  generate(request: $request) {
    apiCreditCost
    generationId
    __typename
  }
}`

// Status query
const statusQuery = `query GetAIGenerationFeedStatuses($where: generations_bool_exp = {}) {
  generations(where: $where) {
    id
    status
    __typename
  }
}`

// Detail query
const detailQuery = `query GetGenerationDetail($where: generations_bool_exp = {}) {
  generations(where: $where) {
    id
    status
    prompt
    modelId
    imageWidth
    imageHeight
    generated_images(order_by: [{url: desc}]) {
      id
      url
      motionMP4URL
      motionGIFURL
      __typename
    }
    __typename
  }
}`

// GenerateSeedance25 submits a Seedance 2.5 video generation request
func GenerateSeedance25(jwtToken, prompt string, width, height, duration int) (string, error) {
	// Build parameters
	params := map[string]interface{}{
		"prompt":           prompt,
		"quantity":         1,
		"duration":         duration,
		"motion_has_audio": false,
		"width":            width,
		"height":           height,
		"seed":             -1,
		"mode":             "RESOLUTION_720",
		"prompt_enhance":   "OFF",
	}

	// Build GraphQL request
	gqlReq := GraphQLRequest{
		OperationName: "Generate",
		Variables: map[string]interface{}{
			"request": map[string]interface{}{
				"model":      "seedance-2.5",
				"public":     false,
				"parameters": params,
			},
		},
		Query: generateMutation,
	}

	// Marshal to JSON
	body, err := json.Marshal(gqlReq)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", GraphQLURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	// Parse response
	var gqlResp struct {
		Data struct {
			Generate struct {
				APICreditCost int    `json:"apiCreditCost"`
				GenerationID  string `json:"generationId"`
			} `json:"generate"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return "", fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}

	fmt.Printf("Generation submitted: ID=%s, Cost=%d tokens\n",
		gqlResp.Data.Generate.GenerationID,
		gqlResp.Data.Generate.APICreditCost)

	return gqlResp.Data.Generate.GenerationID, nil
}

// PollStatus checks if the generation is complete
func PollStatus(jwtToken, generationID string) (string, error) {
	gqlReq := GraphQLRequest{
		OperationName: "GetAIGenerationFeedStatuses",
		Variables: map[string]interface{}{
			"where": map[string]interface{}{
				"id": map[string]interface{}{
					"_in": []string{generationID},
				},
				"status": map[string]interface{}{
					"_in": []string{"PENDING", "COMPLETE", "FAILED"},
				},
			},
		},
		Query: statusQuery,
	}

	body, err := json.Marshal(gqlReq)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", GraphQLURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var gqlResp struct {
		Data struct {
			Generations []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"generations"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return "", fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}

	if len(gqlResp.Data.Generations) == 0 {
		return "UNKNOWN", nil
	}

	return gqlResp.Data.Generations[0].Status, nil
}

// GetVideoURL retrieves the final video URL
func GetVideoURL(jwtToken, generationID string) (string, error) {
	gqlReq := GraphQLRequest{
		OperationName: "GetGenerationDetail",
		Variables: map[string]interface{}{
			"where": map[string]interface{}{
				"id": map[string]interface{}{
					"_in": []string{generationID},
				},
			},
		},
		Query: detailQuery,
	}

	body, err := json.Marshal(gqlReq)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", GraphQLURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var gqlResp struct {
		Data struct {
			Generations []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
				Images []struct {
					URL          string `json:"url"`
					MotionMP4URL string `json:"motionMP4URL"`
				} `json:"generated_images"`
			} `json:"generations"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return "", fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}

	if len(gqlResp.Data.Generations) == 0 {
		return "", fmt.Errorf("generation not found")
	}

	gen := gqlResp.Data.Generations[0]
	if len(gen.Images) == 0 {
		return "", fmt.Errorf("no video generated")
	}

	return gen.Images[0].MotionMP4URL, nil
}

// GenerateVideoComplete - Full workflow: submit, poll, return URL
func GenerateVideoComplete(jwtToken, prompt string, width, height, duration int) (string, error) {
	// Step 1: Submit generation
	fmt.Println("Submitting video generation...")
	generationID, err := GenerateSeedance25(jwtToken, prompt, width, height, duration)
	if err != nil {
		return "", fmt.Errorf("submit generation: %w", err)
	}

	// Step 2: Poll for completion
	fmt.Println("Waiting for video to complete...")
	maxAttempts := 60 // 5 minutes max (60 * 5 seconds)
	for i := 0; i < maxAttempts; i++ {
		time.Sleep(5 * time.Second)

		status, err := PollStatus(jwtToken, generationID)
		if err != nil {
			return "", fmt.Errorf("poll status: %w", err)
		}

		fmt.Printf("Status: %s\n", status)

		if status == "COMPLETE" {
			// Step 3: Get video URL
			videoURL, err := GetVideoURL(jwtToken, generationID)
			if err != nil {
				return "", fmt.Errorf("get video URL: %w", err)
			}

			fmt.Printf("Video ready: %s\n", videoURL)
			return videoURL, nil
		}

		if status == "FAILED" {
			return "", fmt.Errorf("generation failed")
		}
	}

	return "", fmt.Errorf("generation timeout")
}

func main() {
	// Example usage
	jwtToken := "YOUR_JWT_TOKEN_HERE"
	prompt := "a cat running in the park"
	width := 1280
	height := 720
	duration := 4

	videoURL, err := GenerateVideoComplete(jwtToken, prompt, width, height, duration)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Success! Video URL: %s\n", videoURL)
}
