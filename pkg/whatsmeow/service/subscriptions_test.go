package whatsmeow_service

import (
	"reflect"
	"testing"

	"github.com/EvolutionAPI/evolution-go/pkg/internal/event_types"
)

func TestParseEventSubscriptions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string defaults to MESSAGE",
			input: "",
			want:  []string{event_types.MESSAGE},
		},
		{
			name:  "whitespace and commas only defaults to MESSAGE",
			input: " ,  , ",
			want:  []string{event_types.MESSAGE},
		},
		{
			name:  "trims spaces and uppercases events",
			input: " message , connection , qrcode ",
			want:  []string{"MESSAGE", "CONNECTION", "QRCODE"},
		},
		{
			name:  "preserves ALL keyword",
			input: "ALL",
			want:  []string{"ALL"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEventSubscriptions(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseEventSubscriptions(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

