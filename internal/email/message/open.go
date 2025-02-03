package message

import (
	"bytes"
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/email/user"
	"email.mercata.com/internal/utils"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func OpenFromUnsealedHeaders(messageDirPath string) (*Message, error) {
	headersPath := filepath.Join(messageDirPath, MESSAGE_DIR_HEADERS_FILE_NAME)
	headersFileExists, err := utils.FilePathExists(headersPath)
	if err != nil {
		return nil, err
	}
	if !headersFileExists {
		return nil, errors.New("Message is not opened.")
	}
	headersData, err := ioutil.ReadFile(headersPath)
	if err != nil {
		fmt.Println("Error: could not read headers file")
		return nil, err
	}
	var message Message
	err = parseContentHeaders(&message, headersData)
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func cleanupFailedOpen(messageDirPath string) {
	_ = os.Remove(filepath.Join(messageDirPath, MESSAGE_DIR_HEADERS_FILE_NAME))
}

func Open(messageDirPath string, authorUser *user.User, readerUser *user.Reader) (*Message, error) {
	destHeadersPath := filepath.Join(messageDirPath, MESSAGE_DIR_HEADERS_FILE_NAME)

	headersFileExists, err := utils.FilePathExists(destHeadersPath)
	if err != nil {
		return nil, err
	}
	if headersFileExists {
		return nil, errors.New("Message is already opened.")
	}

	message, err := openEnvelopeFile(messageDirPath, authorUser, readerUser)
	if err != nil {
		cleanupFailedOpen(messageDirPath)
		return nil, err
	}

	err = ioutil.WriteFile(destHeadersPath, append(message.ContentHeadersBytes, '\n'), 0644)
	if err != nil {
		cleanupFailedOpen(messageDirPath)
		return nil, err
	}

	var fileInfo *crypto.IOInfo
	payloadFullPath := filepath.Join(messageDirPath, consts.MESSAGE_DIR_PAYLOAD_FILE_NAME)

	destFilePath := filepath.Join(messageDirPath, MESSAGE_DIR_BODY_FILE_NAME)
	if message.IsFile() {
		destFilePath = filepath.Join(messageDirPath, MESSAGE_DIR_FILE_NAME)
	}

	if message.IsBroadcast {
		// Note: We could just rename but then we would not be calculating checksum.
		fileInfo, err = crypto.CopyFile(payloadFullPath, destFilePath)
		if err != nil {
			cleanupFailedOpen(messageDirPath)
			return nil, err
		}
	} else {
		// TODO: Check algorithm
		if message.PayloadCipher.Stream {
			fileInfo, err = crypto.SecretStreamXchacha20Poly1305DecryptFile(payloadFullPath, destFilePath, message.AccessKey, message.PayloadCipher.ChunkSize)
		} else {
			fileInfo, err = crypto.Xchacha20Poly1305DecryptFile(payloadFullPath, destFilePath, message.AccessKey)
		}
		if err != nil {
			cleanupFailedOpen(messageDirPath)
			return nil, err
		}
	}

	if message.Content.Checksum != fileInfo.OutputChecksum {
		fmt.Printf("signed content checksum %s <=> calculated checksum %s", message.Content.Checksum, fileInfo.OutputChecksum)
		return nil, errors.New("Content checksum failure: seal broken")
	}

	if message.IsFile() {
		utils.SetFileModificationTime(destFilePath, message.FileModifiedAt)
	}

	e := os.Remove(payloadFullPath)
	if e != nil {
		return nil, err
	}
	return message, nil
}

func (msg *Message) VerifyEnvelopeAuthenticity() bool {
	if (msg.EnvelopeHeadersOrder == "") || (msg.EnvelopeHeadersChecksum == "") || (msg.EnvelopeHeadersSignature == "") {
		return false
	}

	var buffer bytes.Buffer

	for _, h := range strings.Split(msg.EnvelopeHeadersOrder, ":") {
		h = strings.ToLower(strings.TrimSpace(h))
		switch h {
		case HEADER_MESSAGE_ID:
			buffer.WriteString(msg.ID)

		case HEADER_MESSAGE_STREAM:
			buffer.WriteString(msg.StreamID)

		case HEADER_MESSAGE_ACCESS:
			buffer.WriteString(msg.AccessList)

		case HEADER_MESSAGE_CONTENT_HEADERS:
			buffer.WriteString(msg.ContentHeadersData)

		case HEADER_MESSAGE_ENCRYPTION:
			buffer.WriteString(msg.PayloadCipher.OriginalHeaderValue)

		case HEADER_MESSAGE_ENVELOPE_CHECKSUM:
			continue
		case HEADER_MESSAGE_ENVELOPE_SIGNATURE:
			continue
		default:
			fmt.Printf("WARNING: unknown envelope key in order '%s'\n", h)
			continue
		}
	}

	values := buffer.Bytes()
	strSum, sumBytes := crypto.Checksum(values)

	if msg.EnvelopeHeadersChecksum != strSum {
		return false
	}

	if !crypto.VerifySignature(msg.Author.PublicSigningKey, msg.EnvelopeHeadersSignature, sumBytes) {
		fmt.Println(msg.EnvelopeHeadersSignature)
		return false
	}
	return true
}
