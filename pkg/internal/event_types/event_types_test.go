package event_types

import "testing"

func TestProfileEventsAreValid(t *testing.T) {
	for _, eventType := range []string{PICTURE, USER_ABOUT} {
		if !IsEventType(eventType) {
			t.Errorf("expected %s to be a valid event type", eventType)
		}
	}
}
