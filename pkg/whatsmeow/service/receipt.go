package whatsmeow_service

import (
	"strings"

	"go.mau.fi/whatsmeow/types"
)

// validReceiptMessageIDs removes IDs that cannot be associated with a message
// and collapses duplicates within the same receipt. WhatsApp may omit the key
// attribute in grouped receipt stanzas, which Whatsmeow currently exposes as
// an empty message ID.
func validReceiptMessageIDs(messageIDs []types.MessageID) []types.MessageID {
	validIDs := make([]types.MessageID, 0, len(messageIDs))
	seen := make(map[types.MessageID]struct{}, len(messageIDs))
	for _, messageID := range messageIDs {
		rawID := string(messageID)
		if rawID == "" || strings.TrimSpace(rawID) != rawID {
			continue
		}
		if _, duplicate := seen[messageID]; duplicate {
			continue
		}
		seen[messageID] = struct{}{}
		validIDs = append(validIDs, messageID)
	}
	return validIDs
}
