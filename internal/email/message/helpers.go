package message

import (
	"bufio"
	"bytes"
	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/email/user"
	"email.mercata.com/internal/utils"
	"strings"
)

const ACCESS_LINE_ATTRIBUTE_LINK = "link"
const ACCESS_LINE_ATTRIBUTE_NONCE_KEY_FINGERPRINT = "access-key"
const ACCESS_LINE_ATTRIBUTE_DATA = "value"
const ACCESS_LINE_ATTRIBUTE_ALGORITHM = "algorithm"
const ACCESS_LINE_ATTRIBUTE_FINGERPRINT = "key"

func HeaderLine(key, value string) string {
	return key + HEADER_KEY_VALUE_SEPARATOR + " " + value
}

func HeadersChecksumHeader(checksum, headersOrder string) string {
	return "algorithm=" + crypto.CHECKSUM_ALGORITHM + "; order=" + headersOrder + "; value=" + checksum
}

func ChecksumHeader(checksum string) string {
	return "algorithm=" + crypto.CHECKSUM_ALGORITHM + "; value=" + checksum
}

func SignatureHeader(data string) string {
	return "algorithm=" + crypto.SIGNING_ALGORITHM + "; value=" + data
}

func FileHeader(content Content) string {
	return "name=" + content.FileName + "; type=" + content.FileType + "; modified=" + utils.ToRFC3339String(content.FileModifiedAt)
}

func headerBoolString(value bool) string {
	if value {
		return HEADER_YES_VALUE
	}
	return HEADER_NO_VALUE
}

func headerBoolValue(value string) bool {
	return strings.ToLower(value) == strings.ToLower(HEADER_YES_VALUE)
}

func contentReaderLine(link, address string) string {
	// Format:
	// 		LINK: ADDRESS
	return link + HEADER_ACCESS_FIELDS_SEPARATOR + " " + address
}

// The Signing fingerprint is used to determine the person to serve the message.
// The Encryption fingerprint is needed by reader to determine which encryption pair to use.
func envelopeAccessLine(reader *user.Reader) string {
	return strings.Join([]string{
		strings.Join([]string{ACCESS_LINE_ATTRIBUTE_LINK, reader.Link}, "="),
		strings.Join([]string{ACCESS_LINE_ATTRIBUTE_NONCE_KEY_FINGERPRINT, reader.User.PublicSigningKeyFingerprint}, "="),
		strings.Join([]string{ACCESS_LINE_ATTRIBUTE_DATA, reader.SealedKey}, "="),
		strings.Join([]string{ACCESS_LINE_ATTRIBUTE_ALGORITHM, crypto.ANONYMOUS_ENCRYPTION_CIPHER}, "="),
		strings.Join([]string{ACCESS_LINE_ATTRIBUTE_FINGERPRINT, reader.User.PublicEncryptionKeyFingerprint}, "=")}, "; ")
}

// TODO: generic
func parseEnvelopeAccessLine(line string) *user.Reader {
	attributes := strings.Split(line, ";")

	reader := user.Reader{}

	for _, s := range attributes {
		parts := strings.SplitN(strings.TrimSpace(s), "=", 2)
		switch strings.TrimSpace(parts[0]) {
		case ACCESS_LINE_ATTRIBUTE_LINK:
			reader.Link = strings.TrimSpace(parts[1])
		case ACCESS_LINE_ATTRIBUTE_NONCE_KEY_FINGERPRINT:
			reader.User.PublicSigningKeyFingerprint = strings.TrimSpace(parts[1])
		case ACCESS_LINE_ATTRIBUTE_DATA:
			reader.SealedKey = strings.TrimSpace(parts[1])
		case ACCESS_LINE_ATTRIBUTE_ALGORITHM:
			// Allow only one type of seal
			if strings.ToLower(strings.TrimSpace(parts[1])) != crypto.ANONYMOUS_ENCRYPTION_CIPHER {
				return nil
			}
		case ACCESS_LINE_ATTRIBUTE_FINGERPRINT:
			reader.User.PublicEncryptionKeyFingerprint = strings.TrimSpace(parts[1])
		default:
			continue
		}
	}

	return &reader
}

func LinkFingerprintExistsInAccessList(link, keyFingerprint string, accessListStr *[]byte) (bool, error) {
	headerKey := HEADER_MESSAGE_ACCESS + HEADER_KEY_VALUE_SEPARATOR
	scanner := bufio.NewScanner(bytes.NewReader(*accessListStr))
	for scanner.Scan() {
		envLine := strings.TrimSpace(scanner.Text())
		if strings.ToLower(envLine[:len(headerKey)]) == headerKey {
			accessLines := strings.Split(envLine[len(headerKey):], HEADER_ACCESS_LIST_SEPARATOR)
			for _, access := range accessLines {
				access = strings.TrimSpace(access)
				reader := parseEnvelopeAccessLine(access)
				if (reader.Link == link) && (reader.User.PublicSigningKeyFingerprint == keyFingerprint) {
					return true, nil
				}
			}
			return false, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}
