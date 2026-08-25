package handler

import (
	"strings"
	"testing"

	"leo2api/internal/provider/leonardo"
)

func TestValidateLeonardoReferenceVideoDimensions(t *testing.T) {
	tests := []struct {
		name      string
		width     *int
		height    *int
		wantError string
	}{
		{name: "valid landscape", width: intPtr(1280), height: intPtr(720)},
		{name: "valid portrait", width: intPtr(720), height: intPtr(1280)},
		{name: "too narrow", width: intPtr(480), height: intPtr(854), wantError: "480x854"},
		{name: "too large", width: intPtr(3840), height: intPtr(2160), wantError: "3840x2160"},
		{name: "missing metadata", wantError: "unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLeonardoReferenceVideoDimensions(&leonardo.UploadedMedia{
				Width: tt.width, Height: tt.height,
			})
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestMotionHasAudioFromRequest(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		want bool
	}{
		{name: "default enabled", want: true},
		{name: "disable audio", data: map[string]interface{}{"disable_audio": true}, want: false},
		{name: "explicit enabled", data: map[string]interface{}{"disable_audio": false}, want: true},
		{name: "legacy motion flag", data: map[string]interface{}{"motion_has_audio": false}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := motionHasAudioFromRequest(tt.data); got != tt.want {
				t.Fatalf("motionHasAudioFromRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func intPtr(value int) *int { return &value }
