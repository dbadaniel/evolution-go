package send_service

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestNormalizeLinkPreviewThumbnailLimitsDimensionsAndProducesJPEG(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 1000, 1))
	var input bytes.Buffer
	if err := png.Encode(&input, source); err != nil {
		t.Fatalf("encode source PNG: %v", err)
	}

	thumbnail, width, height, sourceFormat, err := normalizeLinkPreviewThumbnail(input.Bytes(), 200)
	if err != nil {
		t.Fatalf("normalize thumbnail: %v", err)
	}
	if sourceFormat != "png" {
		t.Fatalf("unexpected source format: %q", sourceFormat)
	}
	if width != 200 || height != 1 {
		t.Fatalf("unexpected normalized dimensions: %dx%d", width, height)
	}

	decoded, format, err := image.Decode(bytes.NewReader(thumbnail))
	if err != nil {
		t.Fatalf("decode normalized thumbnail: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("normalized thumbnail must be JPEG, got %q", format)
	}
	if decoded.Bounds().Dx() != int(width) || decoded.Bounds().Dy() != int(height) {
		t.Fatalf("encoded dimensions differ from metadata: %dx%d", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
}

func TestNormalizeLinkPreviewThumbnailInvalidImageFallsBack(t *testing.T) {
	thumbnail, width, height, _, err := normalizeLinkPreviewThumbnail([]byte("not-an-image"), 200)
	if err == nil {
		t.Fatal("expected invalid image error")
	}
	if thumbnail != nil || width != 0 || height != 0 {
		t.Fatalf("invalid image must return empty fallback, got bytes=%d dimensions=%dx%d", len(thumbnail), width, height)
	}
}

func TestBuildLinkExtendedTextMessageWithThumbnail(t *testing.T) {
	data := &LinkStruct{
		Text:        "Confira https://example.com/article",
		Title:       "Example",
		Description: "An example article",
	}
	matchedText := "https://example.com/article"
	thumbnail := []byte{0xff, 0xd8, 0xff, 0xd9}

	message := buildLinkExtendedTextMessage(data, matchedText, thumbnail, 200, 120)

	if message.GetText() != data.Text {
		t.Fatalf("unexpected text: %q", message.GetText())
	}
	if message.GetMatchedText() != matchedText {
		t.Fatalf("unexpected matched URL: %q", message.GetMatchedText())
	}
	if message.GetTitle() != data.Title || message.GetDescription() != data.Description {
		t.Fatalf("metadata was not preserved: title=%q description=%q", message.GetTitle(), message.GetDescription())
	}
	if message.GetPreviewType() != waE2E.ExtendedTextMessage_IMAGE {
		t.Fatalf("unexpected preview type: %s", message.GetPreviewType())
	}
	if !bytes.Equal(message.GetJPEGThumbnail(), thumbnail) {
		t.Fatal("thumbnail was not preserved in JPEGThumbnail")
	}
	if message.GetThumbnailWidth() != 200 || message.GetThumbnailHeight() != 120 {
		t.Fatalf("unexpected thumbnail dimensions: %dx%d", message.GetThumbnailWidth(), message.GetThumbnailHeight())
	}
	if message.GetContextInfo() != nil {
		t.Fatal("common link preview must not include context info or external ad reply")
	}

	encoded, err := proto.Marshal(message)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}
	if bytes.Count(encoded, thumbnail) != 1 {
		t.Fatalf("thumbnail must occur exactly once in protobuf, got %d", bytes.Count(encoded, thumbnail))
	}
}

func TestBuildLinkExtendedTextMessageWithoutThumbnail(t *testing.T) {
	data := &LinkStruct{
		Text:        "Confira https://example.com/article",
		Title:       "Example",
		Description: "An example article",
	}
	matchedText := "https://example.com/article"

	message := buildLinkExtendedTextMessage(data, matchedText, nil, 0, 0)

	if message.GetMatchedText() != matchedText {
		t.Fatalf("unexpected matched URL: %q", message.GetMatchedText())
	}
	if message.GetPreviewType() != waE2E.ExtendedTextMessage_PLACEHOLDER {
		t.Fatalf("unexpected preview type: %s", message.GetPreviewType())
	}
	if len(message.GetJPEGThumbnail()) != 0 {
		t.Fatal("message without an image must not include a thumbnail")
	}
	if message.ThumbnailWidth != nil || message.ThumbnailHeight != nil {
		t.Fatal("message without an image must not include thumbnail dimensions")
	}
	if message.GetContextInfo() != nil {
		t.Fatal("common link preview must not include context info or external ad reply")
	}
}

func TestBuildLinkExtendedTextMessageRejectsThumbnailWithoutDimensions(t *testing.T) {
	data := &LinkStruct{Text: "Confira https://example.com/article"}

	message := buildLinkExtendedTextMessage(data, "https://example.com/article", []byte{0xff, 0xd8}, 0, 120)

	if message.GetPreviewType() != waE2E.ExtendedTextMessage_PLACEHOLDER {
		t.Fatalf("unexpected preview type: %s", message.GetPreviewType())
	}
	if len(message.GetJPEGThumbnail()) != 0 || message.ThumbnailWidth != nil || message.ThumbnailHeight != nil {
		t.Fatal("thumbnail with invalid dimensions must be discarded")
	}
}
