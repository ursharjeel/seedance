package handler

import (
	"encoding/base64"
	"fmt"
	"mime"
	"net/url"
	"strings"
)

// decodeInlineMediaURL decodes a browser data URL while applying the same
// byte limit and media-type validation used by remote media fetches.
//
// Browser uploads commonly arrive as data:[type][;base64],... URLs. They are
// deliberately handled separately from HTTP fetching so existing SSRF and
// remote-fetch behavior remains unchanged.
func decodeInlineMediaURL(rawURL string, maxBytes int, mediaPrefix string) ([]byte, string, string, string, error) {
	value := strings.TrimSpace(rawURL)
	if len(value) < len("data:") || !strings.EqualFold(value[:len("data:")], "data:") {
		return nil, "", "", "", fmt.Errorf("not an inline data url")
	}

	separator := strings.IndexByte(value[len("data:"):], ',')
	if separator < 0 {
		return nil, "", "", "", fmt.Errorf("inline data url is missing a payload")
	}
	separator += len("data:")
	metadata := value[len("data:"):separator]
	payload := value[separator+1:]
	if payload == "" {
		return nil, "", "", "", fmt.Errorf("inline data url has an empty payload")
	}
	if len(metadata) > 1024 {
		return nil, "", "", "", fmt.Errorf("inline data url metadata is too long")
	}

	parts := strings.Split(metadata, ";")
	mediaType := "text/plain"
	if parts[0] != "" {
		mediaType = strings.TrimSpace(parts[0])
	}
	parsedMediaType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("invalid inline media type: %w", err)
	}
	mediaType = strings.ToLower(strings.TrimSpace(parsedMediaType))
	if !strings.HasPrefix(mediaType, strings.ToLower(mediaPrefix)) {
		return nil, "", "", "", fmt.Errorf("inline media type must use %s", mediaPrefix)
	}

	isBase64 := false
	for _, part := range parts[1:] {
		if strings.EqualFold(strings.TrimSpace(part), "base64") {
			isBase64 = true
			break
		}
	}

	var data []byte
	if isBase64 {
		// Reject oversized payloads before allocating the decoded buffer.
		if maxBytes > 0 && base64.StdEncoding.DecodedLen(len(payload)) > maxBytes+2 {
			return nil, "", "", "", fmt.Errorf("inline media exceeds %d MB limit", maxBytes>>20)
		}
		data, err = base64.StdEncoding.DecodeString(payload)
		if err != nil {
			// RawStdEncoding accepts browser payloads that omit optional padding.
			data, err = base64.RawStdEncoding.DecodeString(payload)
		}
		if err != nil {
			return nil, "", "", "", fmt.Errorf("invalid inline base64 payload: %w", err)
		}
	} else {
		var decodedPayload string
		decodedPayload, err = url.PathUnescape(payload)
		if err != nil {
			return nil, "", "", "", fmt.Errorf("invalid inline data payload: %w", err)
		}
		data = []byte(decodedPayload)
	}
	if len(data) == 0 {
		return nil, "", "", "", fmt.Errorf("inline data url has an empty payload")
	}
	if maxBytes > 0 && len(data) > maxBytes {
		return nil, "", "", "", fmt.Errorf("inline media exceeds %d MB limit", maxBytes>>20)
	}

	var ext string
	switch strings.ToLower(strings.TrimSpace(mediaPrefix)) {
	case "image/":
		ext = imageExtFromContentType(mediaType)
	case "video/":
		ext = videoExtFromContentType(mediaType)
	case "audio/":
		ext = audioExtFromContentType(mediaType)
	}
	if ext == "" {
		return nil, "", "", "", fmt.Errorf("inline media type %q is not supported", mediaType)
	}

	return data, mediaType, ext, "media." + ext, nil
}
