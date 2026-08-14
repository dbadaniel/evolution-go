package whatsmeow_service

import (
	"encoding/json"
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

func TestExtractReferralFromMessage(t *testing.T) {
	message := &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
		Text: referralStringPtr("Quero saber mais sobre este anúncio."),
		ContextInfo: &waE2E.ContextInfo{ExternalAdReply: &waE2E.ContextInfo_ExternalAdReplyInfo{
			CtwaClid:          referralStringPtr("FAKE_CLID_abc123xyz"),
			SourceURL:         referralStringPtr("https://fb.me/fake-ad-link"),
			ShowAdAttribution: referralBoolPtr(true),
		}},
	}}

	referral := extractReferralFromMessage(message)
	if len(referral) == 0 {
		t.Fatal("expected referral payload, got empty")
	}
	var got map[string]any
	if err := json.Unmarshal(referral, &got); err != nil {
		t.Fatalf("unmarshal referral: %v", err)
	}
	if got["ctwaClid"] != "FAKE_CLID_abc123xyz" || got["sourceURL"] != "https://fb.me/fake-ad-link" || got["showAdAttribution"] != true {
		t.Fatalf("unexpected referral payload: %#v", got)
	}
}

func TestExtractReferralFromMessageWithoutReferral(t *testing.T) {
	if got := extractReferralFromMessage(&waE2E.Message{}); got != nil {
		t.Fatalf("expected nil referral, got %s", got)
	}
}

func referralStringPtr(value string) *string { return &value }
func referralBoolPtr(value bool) *bool       { return &value }
