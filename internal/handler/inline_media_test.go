package handler

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestDecodeInlineMediaURL(t *testing.T) {
	payload := []byte("RIFF-inline-audio")
	dataURL := "data:audio/wav;base64," + base64.StdEncoding.EncodeToString(payload)

	decoded, contentType, ext, filename, err := decodeInlineMediaURL(dataURL, 1<<20, "audio/")
	if err != nil {
		t.Fatalf("decode inline audio: %v", err)
	}
	if string(decoded) != string(payload) || contentType != "audio/wav" || ext != "wav" || filename != "media.wav" {
		t.Fatalf("decoded inline audio = %q, %q, %q, %q", decoded, contentType, ext, filename)
	}
}

func TestDecodeInlineMediaURLAcceptsUnpaddedBase64AndParameters(t *testing.T) {
	payload := []byte("webm-audio")
	encoded := strings.TrimRight(base64.StdEncoding.EncodeToString(payload), "=")
	decoded, contentType, ext, _, err := decodeInlineMediaURL("DATA:audio/webm;codecs=opus;base64,"+encoded, 1<<20, "audio/")
	if err != nil {
		t.Fatalf("decode unpadded inline audio: %v", err)
	}
	if string(decoded) != string(payload) || contentType != "audio/webm" || ext != "webm" {
		t.Fatalf("decoded unpadded inline audio = %q, %q, %q", decoded, contentType, ext)
	}
}

func TestDecodeInlineMediaURLPercentEncodedPayload(t *testing.T) {
	decoded, contentType, ext, _, err := decodeInlineMediaURL("data:audio/mpeg,%00%01%FF", 1<<20, "audio/")
	if err != nil {
		t.Fatalf("decode percent-encoded inline audio: %v", err)
	}
	if len(decoded) != 3 || decoded[0] != 0 || decoded[1] != 1 || decoded[2] != 0xff || contentType != "audio/mpeg" || ext != "mp3" {
		t.Fatalf("decoded percent-encoded inline audio = %v, %q, %q", decoded, contentType, ext)
	}
}

func TestDecodeInlineMediaURLRejectsInvalidOrWrongMedia(t *testing.T) {
	tests := []string{
		"data:audio/wav;base64,not-valid-base64!",
		"data:image/png;base64,aW1hZ2U=",
		"data:audio/wav;base64,",
		"data:audio/wav;base64,QUJD",
	}
	for _, input := range tests {
		if _, _, _, _, err := decodeInlineMediaURL(input, 2, "audio/"); err == nil {
			t.Errorf("decodeInlineMediaURL(%q) unexpectedly succeeded", input)
		}
	}
}
