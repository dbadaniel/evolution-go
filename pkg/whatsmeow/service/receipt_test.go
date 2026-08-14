package whatsmeow_service

import (
	"reflect"
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestValidReceiptMessageIDs(t *testing.T) {
	tests := []struct {
		name       string
		messageIDs []types.MessageID
		want       []types.MessageID
	}{
		{
			name: "nil list",
			want: []types.MessageID{},
		},
		{
			name:       "empty and malformed whitespace IDs",
			messageIDs: []types.MessageID{"", "   ", " 3EB0ABC", "3EB0DEF\t"},
			want:       []types.MessageID{},
		},
		{
			name:       "valid IDs",
			messageIDs: []types.MessageID{"3EB0ABC", "3EB0DEF"},
			want:       []types.MessageID{"3EB0ABC", "3EB0DEF"},
		},
		{
			name:       "mixed IDs",
			messageIDs: []types.MessageID{"", "3EB0ABC", "\t", "3EB0DEF"},
			want:       []types.MessageID{"3EB0ABC", "3EB0DEF"},
		},
		{
			name:       "duplicate IDs",
			messageIDs: []types.MessageID{"3EB0ABC", "3EB0ABC", "3EB0DEF"},
			want:       []types.MessageID{"3EB0ABC", "3EB0DEF"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validReceiptMessageIDs(tt.messageIDs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("validReceiptMessageIDs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
