package send_service

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/chai2010/webp"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestImageDimensionsSurviveMessageSerialization(t *testing.T) {
	cases := []struct {
		name          string
		width, height int
		encode        func(*bytes.Buffer, image.Image) error
	}{
		{"wide JPEG", 1600, 720, func(out *bytes.Buffer, img image.Image) error {
			return jpeg.Encode(out, img, nil)
		}},
		{"portrait PNG", 240, 640, func(out *bytes.Buffer, img image.Image) error {
			return png.Encode(out, img)
		}},
		{"square PNG", 80, 80, func(out *bytes.Buffer, img image.Image) error {
			return png.Encode(out, img)
		}},
		{"wide WebP", 480, 160, func(out *bytes.Buffer, img image.Image) error {
			return webp.Encode(out, img, &webp.Options{Lossless: true})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var media bytes.Buffer
			if err := tc.encode(&media, image.NewRGBA(image.Rect(0, 0, tc.width, tc.height))); err != nil {
				t.Fatal(err)
			}
			message := &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
				Caption:       proto.String("Original image"),
				JPEGThumbnail: []byte("separate thumbnail"),
			}}
			populateImageDimensions(message.ImageMessage, media.Bytes())
			encoded, err := proto.Marshal(message)
			if err != nil {
				t.Fatal(err)
			}
			decoded := &waE2E.Message{}
			if err := proto.Unmarshal(encoded, decoded); err != nil {
				t.Fatal(err)
			}
			imageMessage := decoded.GetImageMessage()
			if imageMessage.Width == nil || imageMessage.Height == nil || imageMessage.GetWidth() != uint32(tc.width) || imageMessage.GetHeight() != uint32(tc.height) {
				t.Fatalf("serialized image dimensions = %dx%d, want original %dx%d", imageMessage.GetWidth(), imageMessage.GetHeight(), tc.width, tc.height)
			}
			if imageMessage.GetCaption() != "Original image" || !bytes.Equal(imageMessage.GetJPEGThumbnail(), message.ImageMessage.GetJPEGThumbnail()) {
				t.Fatal("adding dimensions changed the caption or thumbnail")
			}
		})
	}
}

func TestImageDimensionsDecodeFailurePreservesMetadata(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("invalid image"), {0xff, 0xd8, 0xff}} {
		message := &waE2E.ImageMessage{}
		populateImageDimensions(message, data)
		if message.Width != nil || message.Height != nil {
			t.Fatal("undecodable image gained dimension fields")
		}
		message.Width, message.Height = proto.Uint32(640), proto.Uint32(480)
		populateImageDimensions(message, data)
		if message.GetWidth() != 640 || message.GetHeight() != 480 {
			t.Fatal("decode failure overwrote existing dimensions")
		}
	}
	populateImageDimensions(nil, nil)
}
