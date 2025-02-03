package message

import (
	"bufio"
	"bytes"
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/email/address"
	linksPkg "email.mercata.com/internal/email/links"
	"email.mercata.com/internal/email/user"
	"email.mercata.com/internal/utils"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func openContentHeaders(message *Message, contentHeadersBytes string) ([]byte, error) {
	headers, err := crypto.Xchacha20Poly1305DecodeDecrypt(contentHeadersBytes, message.AccessKey)
	if err != nil {
		return nil, err
	}
	return headers, nil
}

func parseContentHeaders(message *Message, contentHeaders []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(contentHeaders))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line[0] == '#' {
			continue
		}

		parts := strings.SplitN(line, HEADER_KEY_VALUE_SEPARATOR, 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case HEADER_CONTENT_MESSAGE_ID:
			message.Content.ID = value
			if message.ID == "" && utils.ValidMessageID(value) {
				message.ID = value
			}

		case HEADER_CONTENT_AUTHOR:
			message.Content.AuthorAddress = value
			if message.Author.Address == "" {
				address, domain, localPart := address.ParseEmailAddress(value)
				message.Author.Address = address
				message.Author.Domain = domain
				message.Author.LocalPart = localPart
			}

		case HEADER_CONTENT_DATE:
			time, err := utils.ParseRFC3339Time(value)
			if err != nil {
				return err
			}
			message.Content.Date = *time

		case HEADER_CONTENT_SIZE:
			size, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			message.Content.Size = size

		case HEADER_CONTENT_CHECKSUM:
			checksumAttrs := utils.ParseHeadersAttributes(value)
			if checksumAttrs["algorithm"] != crypto.CHECKSUM_ALGORITHM {
				return errors.New("unsupported checksum algorithm")
			}
			message.Content.Checksum = checksumAttrs["sum"]

		case HEADER_CONTENT_FILE:
			fileAttrs := utils.ParseHeadersAttributes(value)
			if fileName, ok := fileAttrs["name"]; ok {
				message.Content.FileName = fileName
			} else {
				message.Content.FileName = "unnamed-file"
			}
			if fileType, ok := fileAttrs["type"]; ok {
				message.Content.FileType = fileType
			}
			if fileMod, ok := fileAttrs["modified"]; ok {
				time, err := utils.ParseRFC3339Time(fileMod)
				if err != nil {
					return err
				}
				message.Content.FileModifiedAt = *time
			}

		case HEADER_CONTENT_SUBJECT:
			message.Content.Subject = value

		case HEADER_CONTENT_SUBJECT_ID:
			message.Content.SubjectID = value

		case HEADER_CONTENT_PARENT_MESSAGE_ID:
			message.Content.ParentMessageID = value

		case HEADER_CONTENT_CATEGORY:
			message.Content.Category = value

		case HEADER_CONTENT_READERS:
			message.Content.ReadersAddresses = value

		default:
			fmt.Printf("WARNING: unknown content header key '%s'\n", key)
		}
	}

	// Expand message readers with actual addresses
	contentReaders := strings.Split(message.Content.ReadersAddresses, ",")
	if len(contentReaders) != len(message.Readers) {
		fmt.Println("WARNING: disclosed readers do not match access list")
	}
	for _, address := range contentReaders {
		address = strings.TrimSpace(address)
		link := linksPkg.Make(address, message.Author.Address)
		for i, r := range message.Readers {
			if r.Link == link {
				message.Readers[i].Address = address
				break
			}
		}
	}

	return nil
}

func extractReaderAccess(accessListStr string) []user.Reader {
	accessList := strings.Split(accessListStr, HEADER_ACCESS_LIST_SEPARATOR)
	var readers []user.Reader
	for _, accessStr := range accessList {
		reader := parseEnvelopeAccessLine(strings.TrimSpace(accessStr))
		if reader != nil {
			readers = append(readers, *reader)
		}
	}
	return readers
}

func retrieveAccessKeyForReaderUser(message *Message, readerUser *user.Reader) ([]byte, error) {
	link := linksPkg.Make(message.Author.Address, readerUser.Address)
	for _, r := range message.Readers {
		if r.Link == link && r.PublicEncryptionKeyFingerprint == readerUser.PublicEncryptionKeyFingerprint {
			key, err := crypto.DecryptAnonymous(readerUser.PrivateEncryptionKey, readerUser.PublicEncryptionKey, r.SealedKey)
			if err != nil {
				return nil, err
			}
			return key, nil
		}
	}
	return nil, errors.New("non-designated reader or public key mismatch")
}

func ParseEnvelopeFile(messageDirPath string) (*Message, error) {
	envelopeFullPath := filepath.Join(messageDirPath, consts.MESSAGE_DIR_ENVELOPE_FILE_NAME)
	envelopeData, err := ioutil.ReadFile(envelopeFullPath)
	if err != nil {
		return nil, err
	}
	return ParseEnvelopeData(envelopeData)
}

func ParseEnvelopeData(data []byte) (*Message, error) {
	message := Message{
		IsBroadcast: true,
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line[0] == '#' {
			continue
		}

		parts := strings.SplitN(line, HEADER_KEY_VALUE_SEPARATOR, 2)
		if len(parts) != 2 {
			continue
		}

		_, err := AssignMessageHeader(parts[0], parts[1], &message)
		if err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &message, nil
}

func MessageFromHeadersData(h http.Header) (*Message, error) {
	var headersList []string
	message := Message{IsBroadcast: true}

	for key, values := range h {
		// Values we are interested in can be only singular
		if len(values) > 1 {
			continue
		}
		isMailHeader, err := AssignMessageHeader(key, values[0], &message)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		if isMailHeader {
			headersList = append(headersList, HeaderLine(key, values[0]))
		}
	}
	if message.ID == "" {
		return nil, errors.New("bad message data")
	}

	message.EnvelopeHeadersList = headersList
	return &message, nil
}

func AssignMessageHeader(key, value string, message *Message) (bool, error) {
	value = strings.TrimSpace(value)
	switch strings.ToLower(strings.TrimSpace(key)) {
	case HEADER_MESSAGE_ID:
		message.ID = value

	case HEADER_MESSAGE_STREAM:
		message.StreamID = value

	case HEADER_MESSAGE_ACCESS:
		message.AccessList = value
		message.IsBroadcast = false
		message.Readers = extractReaderAccess(value)

	case HEADER_MESSAGE_CONTENT_HEADERS:
		message.ContentHeadersData = value

	case HEADER_MESSAGE_ENVELOPE_CHECKSUM:
		message.EnvelopeHeadersChecksumUnparsed = value

		checksumAttrMap := utils.ParseHeadersAttributes(value)
		if checksumAttrMap["algorithm"] != crypto.CHECKSUM_ALGORITHM {
			return false, errors.New("unsupported checksum algorithm")
		}
		message.EnvelopeHeadersChecksum = checksumAttrMap["sum"]
		message.EnvelopeHeadersOrder = checksumAttrMap["order"]

	case HEADER_MESSAGE_ENVELOPE_SIGNATURE:
		message.EnvelopeHeadersSignatureUnparsed = value

		signatureAttrMap := utils.ParseHeadersAttributes(value)
		if signatureAttrMap["algorithm"] != crypto.SIGNING_ALGORITHM {
			return false, errors.New("unsupported signing algorithm")
		}
		message.EnvelopeHeadersSignature = signatureAttrMap["data"]

	case HEADER_MESSAGE_ENCRYPTION:
		ci, err := crypto.CipherInfoFromHeader(value)
		if err != nil {
			return true, err
		}
		message.PayloadCipher = ci

	default:
		// Ignore other, unknown keys
		return false, nil
	}
	return true, nil
}

func openEnvelopeFile(messageDirPath string, authorUser *user.User, readerUser *user.Reader) (*Message, error) {
	message, err := ParseEnvelopeFile(messageDirPath)
	if err != nil {
		return nil, err
	}
	message.Author = *authorUser

	if message.ContentHeadersData == "" {
		return nil, errors.New("Invalid message")
	}

	contentHeaderAttrs := strings.Split(message.ContentHeadersData, ";")
	contentHeadersBase64 := ""
	for _, pair := range contentHeaderAttrs {
		kvs := strings.SplitN(pair, "=", 2)
		switch strings.ToLower(strings.TrimSpace(kvs[0])) {
		case "seal":
			algorithm := strings.ToLower(strings.TrimSpace(kvs[1]))
			if algorithm == "none" {
				continue
			}
			if algorithm != crypto.SYMMETRIC_CIPHER {
				return nil, errors.New("Unsupported content headers cipher: " + algorithm)
			}
		case "data":
			contentHeadersBase64 = kvs[1]
		default:
			continue
		}
	}

	if message.IsBroadcast {
		contentHeadersBytes, err := base64.StdEncoding.DecodeString(contentHeadersBase64)
		if err != nil {
			return nil, err
		}
		message.ContentHeadersBytes = contentHeadersBytes
	} else {
		key, err := retrieveAccessKeyForReaderUser(message, readerUser)
		if err != nil {
			return nil, err
		}
		message.AccessKey = key

		decrypted, err := openContentHeaders(message, contentHeadersBase64)
		if err != nil {
			return nil, err
		}
		message.ContentHeadersBytes = decrypted
	}

	if !message.VerifyEnvelopeAuthenticity() {
		return nil, errors.New("Message authenticity failure")
	}

	err = parseContentHeaders(message, message.ContentHeadersBytes)
	if err != nil {
		return nil, err
	}

	return message, nil
}
