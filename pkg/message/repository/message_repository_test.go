package message_repository

import (
	"encoding/json"
	"reflect"
	"testing"

	message_model "github.com/EvolutionAPI/evolution-go/pkg/message/model"
)

func TestMessageUpdateColumnsPreservesExistingReferral(t *testing.T) {
	withoutReferral := messageUpdateColumns(message_model.Message{})
	wantWithout := []string{"timestamp", "status", "source"}
	if !reflect.DeepEqual(withoutReferral, wantWithout) {
		t.Fatalf("columns without referral = %v, want %v", withoutReferral, wantWithout)
	}

	withReferral := messageUpdateColumns(message_model.Message{Referral: json.RawMessage(`{"ctwaClid":"abc"}`)})
	wantWith := []string{"timestamp", "status", "source", "referral"}
	if !reflect.DeepEqual(withReferral, wantWith) {
		t.Fatalf("columns with referral = %v, want %v", withReferral, wantWith)
	}
}
