package whatsmeow_service

import (
	"testing"
)

func TestExtractDeviceID(t *testing.T) {
	tests := []struct {
		name     string
		senderID string
		want     int
	}{
		{
			name:     "Companion device 12 with @s.whatsapp.net",
			senderID: "5511911455505:12@s.whatsapp.net",
			want:     12,
		},
		{
			name:     "Companion device 1 with @s.whatsapp.net",
			senderID: "5511984153021:1@s.whatsapp.net",
			want:     1,
		},
		{
			name:     "Companion device 3 with @lid",
			senderID: "57303604711512:3@lid",
			want:     3,
		},
		{
			name:     "Primary phone without device index",
			senderID: "5511984153021@s.whatsapp.net",
			want:     0,
		},
		{
			name:     "Primary phone LID without device index",
			senderID: "57303604711512@lid",
			want:     0,
		},
		{
			name:     "Empty string",
			senderID: "",
			want:     0,
		},
		{
			name:     "Malformed string without at sign",
			senderID: "5511984153021:12",
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDeviceID(tt.senderID)
			if got != tt.want {
				t.Errorf("extractDeviceID(%q) = %v, want %v", tt.senderID, got, tt.want)
			}
		})
	}
}

func TestCleanSenderID(t *testing.T) {
	tests := []struct {
		name     string
		senderID string
		want     string
	}{
		{
			name:     "Strips :12 from @s.whatsapp.net",
			senderID: "5511911455505:12@s.whatsapp.net",
			want:     "5511911455505@s.whatsapp.net",
		},
		{
			name:     "Leaves primary @s.whatsapp.net unchanged",
			senderID: "5511984153021@s.whatsapp.net",
			want:     "5511984153021@s.whatsapp.net",
		},
		{
			name:     "Strips :3 from @lid",
			senderID: "57303604711512:3@lid",
			want:     "57303604711512@lid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanSenderID(tt.senderID)
			if got != tt.want {
				t.Errorf("cleanSenderID(%q) = %v, want %v", tt.senderID, got, tt.want)
			}
		})
	}
}
