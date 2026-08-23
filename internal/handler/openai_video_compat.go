package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// HandleOpenAIVideo exposes the newer OpenAI Videos API paths while keeping
// /v1/video/generations as the canonical Leo2API implementation.
func (s *Server) HandleOpenAIVideo(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/v1/videos" {
		s.handleOpenAIVideoSubmit(w, r)
		return
	}

	remainder := strings.TrimPrefix(r.URL.Path, "/v1/videos/")
	if remainder == "" || strings.Contains(remainder, "/") && !strings.HasSuffix(remainder, "/content") {
		writeJSON(w, http.StatusBadRequest, errorResp("video id is required", "invalid_request_error"))
		return
	}
	if strings.HasSuffix(remainder, "/content") {
		s.handleOpenAIVideoContent(w, r, strings.TrimSuffix(remainder, "/content"))
		return
	}

	cloned := r.Clone(r.Context())
	cloned.URL.Path = "/v1/video/generations/" + remainder
	recorder := newBufferedHTTPResponse()
	s.HandleVideoGenerationStatus(recorder, cloned)
	writeOpenAIVideoResponse(w, recorder, false)
}

func (s *Server) handleOpenAIVideoSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body", "invalid_request_error"))
		return
	}
	var payload map[string]interface{}
	if json.Unmarshal(body, &payload) == nil {
		if _, exists := payload["duration"]; !exists {
			if seconds, ok := openAIVideoSeconds(payload["seconds"]); ok {
				payload["duration"] = seconds
			}
		}
		if translated, marshalErr := json.Marshal(payload); marshalErr == nil {
			body = translated
		}
	}

	cloned := r.Clone(r.Context())
	cloned.URL.Path = "/v1/video/generations"
	cloned.Body = io.NopCloser(bytes.NewReader(body))
	cloned.ContentLength = int64(len(body))
	recorder := newBufferedHTTPResponse()
	s.HandleVideoGeneration(recorder, cloned)
	writeOpenAIVideoResponse(w, recorder, true)
}

func (s *Server) handleOpenAIVideoContent(w http.ResponseWriter, r *http.Request, generationID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.requireAPIKey(r); err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResp("invalid api key", "authentication_error"))
		return
	}
	if s.ReqLog == nil || strings.TrimSpace(generationID) == "" {
		writeJSON(w, http.StatusNotFound, errorResp("video not found", "not_found_error"))
		return
	}
	entry, ok := s.ReqLog.FindByGenerationID(generationID)
	if !ok || strings.ToUpper(strings.TrimSpace(entry.TaskStatus)) != "COMPLETE" || entry.PreviewURL == "" {
		writeJSON(w, http.StatusNotFound, errorResp("video content is not available", "not_found_error"))
		return
	}
	parsed, err := url.Parse(entry.PreviewURL)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResp("video content is not available", "not_found_error"))
		return
	}
	fileName := filepath.Base(parsed.Path)
	filePath := filepath.Join(s.GeneratedDir, fileName)
	info, err := os.Stat(filePath)
	if err != nil || !info.Mode().IsRegular() {
		writeJSON(w, http.StatusNotFound, errorResp("video content is not available", "not_found_error"))
		return
	}
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, filePath)
}

func openAIVideoSeconds(value interface{}) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), v > 0
	case string:
		seconds, err := strconv.Atoi(strings.TrimSpace(v))
		return seconds, err == nil && seconds > 0
	default:
		return 0, false
	}
}

type bufferedHTTPResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newBufferedHTTPResponse() *bufferedHTTPResponse {
	return &bufferedHTTPResponse{header: make(http.Header), status: http.StatusOK}
}

func (w *bufferedHTTPResponse) Header() http.Header { return w.header }
func (w *bufferedHTTPResponse) WriteHeader(statusCode int) {
	if w.status == http.StatusOK {
		w.status = statusCode
	}
}
func (w *bufferedHTTPResponse) Write(data []byte) (int, error) { return w.body.Write(data) }

func writeOpenAIVideoResponse(w http.ResponseWriter, response *bufferedHTTPResponse, _ bool) {
	for key, values := range response.header {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	body := response.body.Bytes()
	var payload map[string]interface{}
	if json.Unmarshal(body, &payload) == nil {
		if payload["status"] == "succeeded" {
			payload["status"] = "completed"
		}
		if rewritten, err := json.Marshal(payload); err == nil {
			body = append(rewritten, '\n')
		}
	}
	status := response.status
	if status == http.StatusAccepted {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
