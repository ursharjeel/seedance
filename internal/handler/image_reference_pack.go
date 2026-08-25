package handler

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"math"
	"strings"

	"leo2api/internal/provider/leonardo"
)

const (
	// Leonardo deployments commonly reject requests containing more than four
	// image_reference entries. Packing keeps the public API useful without
	// sending an oversized guidance array upstream.
	providerImageReferenceLimit = 4
	maxPublicImageReferences    = 9
	packedImageGroupSize        = 3
	packedImageTileSize         = 512
)

type imageReferenceSource struct {
	URL      string
	Strength string
}

// maybePackOpenAIImageReferences converts 5-9 URL references into a small
// number of contact-sheet references. It returns the original map when no
// packing is needed. A cloned map is used so request validation/debug logging
// still sees the caller's original payload.
func (s *Server) maybePackOpenAIImageReferences(data map[string]interface{}, session *leonardo.TokenSession, modelID string) (map[string]interface{}, error) {
	if data == nil || session == nil || isSora2ModelID(modelID) {
		return data, nil
	}

	sources := collectImageReferenceSources(data)
	if len(sources) <= providerImageReferenceLimit {
		return data, nil
	}
	if len(sources) > maxPublicImageReferences {
		return nil, fmt.Errorf("at most %d image references are supported", maxPublicImageReferences)
	}

	packed, err := s.packImageReferenceSources(session, sources)
	if err != nil {
		return nil, fmt.Errorf("pack image references: %w", err)
	}

	cloned := cloneImageReferencePayload(data)
	delete(cloned, "image_url")
	delete(cloned, "image_urls")
	delete(cloned, "image_guidance")
	delete(cloned, "image_reference")
	if guidances, ok := cloned["guidances"].(map[string]interface{}); ok {
		guidances = cloneImageReferencePayload(guidances)
		delete(guidances, "image_reference")
		cloned["guidances"] = guidances
	}

	entries := make([]interface{}, 0, len(packed))
	for _, ref := range packed {
		entries = append(entries, map[string]interface{}{
			"id":       ref.ID,
			"type":     ref.Type,
			"strength": ref.Strength,
		})
	}
	cloned["image_guidance"] = entries
	return cloned, nil
}

func cloneImageReferencePayload(source map[string]interface{}) map[string]interface{} {
	cloned := make(map[string]interface{}, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func collectImageReferenceSources(data map[string]interface{}) []imageReferenceSource {
	if data == nil {
		return nil
	}
	var sources []imageReferenceSource
	appendURL := func(raw interface{}) {
		if url := strings.TrimSpace(toString(raw)); url != "" {
			sources = append(sources, imageReferenceSource{URL: url, Strength: "MID"})
		}
	}

	appendURL(data["image_url"])
	if rawURLs, ok := data["image_urls"].([]interface{}); ok {
		for _, rawURL := range rawURLs {
			appendURL(rawURL)
		}
	}
	if rawGuidance, ok := data["image_guidance"].([]interface{}); ok {
		for _, item := range rawGuidance {
			if source, ok := imageReferenceSourceFromEntry(item); ok {
				sources = append(sources, source)
			}
		}
	}
	for _, item := range minimaxH3ImageReferenceInputs(data) {
		if source, ok := imageReferenceSourceFromEntry(item); ok {
			sources = append(sources, source)
		}
	}
	return sources
}

func imageReferenceSourceFromEntry(item interface{}) (imageReferenceSource, bool) {
	entry, ok := item.(map[string]interface{})
	if !ok {
		return imageReferenceSource{}, false
	}
	strength := strings.ToUpper(strings.TrimSpace(toString(entry["strength"])))
	if imageValue, ok := entry["image"].(map[string]interface{}); ok {
		entry = imageValue
	}
	url := strings.TrimSpace(toString(entry["url"]))
	id := strings.TrimSpace(toString(entry["id"]))
	if url == "" && id == "" {
		return imageReferenceSource{}, false
	}
	if strength == "" {
		strength = "MID"
	}
	// An empty URL is retained so packImageReferenceSources can return a
	// precise error for ID-only references rather than silently dropping one.
	return imageReferenceSource{URL: url, Strength: strength}, true
}

func (s *Server) packImageReferenceSources(session *leonardo.TokenSession, sources []imageReferenceSource) ([]leonardo.ImageRef, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("image references are empty")
	}
	for _, source := range sources {
		if strings.TrimSpace(source.URL) == "" {
			return nil, fmt.Errorf("packing requires URL-based image references; ID-only references cannot be downloaded")
		}
	}

	refs := make([]leonardo.ImageRef, 0, int(math.Ceil(float64(len(sources))/packedImageGroupSize)))
	for start := 0; start < len(sources); start += packedImageGroupSize {
		end := start + packedImageGroupSize
		if end > len(sources) {
			end = len(sources)
		}
		composite, err := s.composeImageReferenceGroup(sources[start:end])
		if err != nil {
			return nil, err
		}
		imageID, err := s.uploadLeonardoImageBytes(session, composite, "png", "image/png")
		if err != nil {
			return nil, fmt.Errorf("upload composite reference %d: %w", len(refs)+1, err)
		}
		refs = append(refs, leonardo.ImageRef{ID: imageID, Type: "UPLOADED", Strength: "MID"})
	}
	return refs, nil
}

func (s *Server) composeImageReferenceGroup(sources []imageReferenceSource) ([]byte, error) {
	if len(sources) == 0 || len(sources) > packedImageGroupSize {
		return nil, fmt.Errorf("invalid image reference group size %d", len(sources))
	}

	decoded := make([]image.Image, 0, len(sources))
	for _, source := range sources {
		data, _, _, err := s.downloadRemoteImage(source.URL)
		if err != nil {
			return nil, fmt.Errorf("download %s: %w", source.URL, err)
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w (JPEG, PNG, and GIF are supported for packing)", source.URL, err)
		}
		decoded = append(decoded, img)
	}

	columns := 2
	if len(decoded) == 1 {
		columns = 1
	}
	rows := (len(decoded) + columns - 1) / columns
	canvas := image.NewRGBA(image.Rect(0, 0, columns*packedImageTileSize, rows*packedImageTileSize))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	for index, source := range decoded {
		tile := fitImageReferenceTile(source, packedImageTileSize, packedImageTileSize)
		x := (index % columns) * packedImageTileSize
		y := (index / columns) * packedImageTileSize
		draw.Draw(canvas, image.Rect(x, y, x+packedImageTileSize, y+packedImageTileSize), tile, image.Point{}, draw.Src)
	}

	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, fmt.Errorf("encode composite image: %w", err)
	}
	return output.Bytes(), nil
}

func fitImageReferenceTile(source image.Image, width, height int) *image.RGBA {
	tile := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(tile, tile.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	bounds := source.Bounds()
	sourceWidth := bounds.Dx()
	sourceHeight := bounds.Dy()
	if sourceWidth <= 0 || sourceHeight <= 0 {
		return tile
	}

	scale := math.Min(float64(width)/float64(sourceWidth), float64(height)/float64(sourceHeight))
	destWidth := maxInt(1, int(math.Round(float64(sourceWidth)*scale)))
	destHeight := maxInt(1, int(math.Round(float64(sourceHeight)*scale)))
	offsetX := (width - destWidth) / 2
	offsetY := (height - destHeight) / 2
	for y := 0; y < destHeight; y++ {
		for x := 0; x < destWidth; x++ {
			sourceX := bounds.Min.X + x*sourceWidth/destWidth
			sourceY := bounds.Min.Y + y*sourceHeight/destHeight
			tile.Set(offsetX+x, offsetY+y, source.At(sourceX, sourceY))
		}
	}
	return tile
}
