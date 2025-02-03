package utils

import (
	"email.mercata.com/internal/crypto"
	"strings"
)

const MESSAGE_ID_MINIMUM_LENGTH = 32
const MESSAGE_ID_MAXIMUM_LENGTH = 128

// NOTE: MessageId does not have to be private on author server as it is local user.
// Once the message is fetched by the reader, it is own secrecy issue.
// The actual format does not matter.
// MessageID must be URL friendly, meaning no special encoding required
func NewMessageID(domain, localPart string) (*string, error) {
	randomStr, err := crypto.GenerateRandomString(24)
	if err != nil {
		return nil, err
	}
	msgRawId := []byte(randomStr + domain + localPart)
	messageId, _ := crypto.Sha256Hash(msgRawId)
	return &messageId, nil
}

// Minimum length is 32 characters
// Maximum length is 128 characters
// No URL unsafe characters, only alphanumerics
func ValidMessageID(msgID string) bool {
	msgID = strings.TrimSpace(msgID)
	if msgID == "" || len(msgID) > MESSAGE_ID_MAXIMUM_LENGTH || len(msgID) < MESSAGE_ID_MINIMUM_LENGTH {
		return false
	}
	return StringIsAlphaNumeric(msgID)
}
