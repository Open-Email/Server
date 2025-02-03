package message

import (
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/email/storage"
	"email.mercata.com/internal/utils"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (msg Message) SealedAccessList() (string, error) {
	var readersList []string
	var err error

	sort.Slice(msg.Readers, func(i, j int) bool {
		return msg.Readers[i].Link < msg.Readers[j].Link
	})

	for i, _ := range msg.Readers {
		r := msg.Readers[i]
		r.SealedKey, err = crypto.EncryptAnonymous(r.PublicEncryptionKey, msg.AccessKey)
		if err != nil {
			return "", err
		}
		readersList = append(readersList, envelopeAccessLine(&r))
	}
	return strings.Join(readersList, HEADER_ACCESS_LIST_SEPARATOR+" "), nil
}

func (msg *Message) SealBody(bodyDestinationPath string) error {
	if msg.IsBroadcast {
		if msg.IsFile() {
			fileInfo, err := crypto.CopyFile(msg.FilePath, bodyDestinationPath)
			if err != nil {
				return err
			}
			msg.Content.Checksum = fileInfo.OutputChecksum
			msg.Content.Size = fileInfo.OutputSize
			return nil
		}
		plainBroadcastFile, err := os.Create(bodyDestinationPath)
		if err != nil {
			return err
		}
		defer plainBroadcastFile.Close()
		_, err = plainBroadcastFile.Write([]byte(msg.Content.Body))
		if err != nil {
			return err
		}
		return nil
	}

	if msg.IsFile() {
		fileInfo, err := crypto.SecretStreamXchacha20Poly1305EncryptFile(msg.FilePath, bodyDestinationPath, msg.AccessKey, msg.PayloadCipher.ChunkSize)
		if err != nil {
			return err
		}
		msg.Content.Checksum = fileInfo.InputChecksum
		msg.Content.Size = fileInfo.InputSize
		return nil
	}

	// the whole contents is encrypted in memory
	encryptedBody, err := crypto.Xchacha20Poly1305Encrypt(msg.Body, msg.AccessKey)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(bodyDestinationPath, encryptedBody, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (msg *Message) EmbedContentHeaders() []byte {
	var headers []string
	// The two fields, MessageID and Author are transferred from the envelope
	headers = append(headers, HeaderLine(HEADER_CONTENT_MESSAGE_ID, msg.ID))
	headers = append(headers, HeaderLine(HEADER_CONTENT_AUTHOR, msg.Author.Address))

	headers = append(headers, HeaderLine(HEADER_CONTENT_DATE, utils.ToRFC3339String(msg.Content.Date)))
	headers = append(headers, HeaderLine(HEADER_CONTENT_SUBJECT, msg.Content.Subject))
	headers = append(headers, HeaderLine(HEADER_CONTENT_SUBJECT_ID, msg.Content.SubjectID))
	headers = append(headers, HeaderLine(HEADER_CONTENT_PARENT_MESSAGE_ID, msg.Content.ParentMessageID))
	headers = append(headers, HeaderLine(HEADER_CONTENT_CATEGORY, msg.Content.Category))
	headers = append(headers, HeaderLine(HEADER_CONTENT_SIZE, strconv.FormatInt(msg.Content.Size, 10)))
	headers = append(headers, HeaderLine(HEADER_CONTENT_CHECKSUM, ChecksumHeader(msg.Content.Checksum)))

	if msg.IsFile() {
		headers = append(headers, HeaderLine(HEADER_CONTENT_FILE, FileHeader(msg.Content)))
	}

	if !msg.IsBroadcast {
		var disclosedReaders []string
		for _, reader := range msg.Readers {
			disclosedReaders = append(disclosedReaders, reader.Address)
		}
		headers = append(headers, HeaderLine(HEADER_CONTENT_READERS, strings.Join(disclosedReaders, HEADER_CONTENT_READERS_ADDRESS_SEPARATOR+" ")))
	}

	return []byte(strings.Join(headers, "\n"))
}

func (msg *Message) SealEnvelope(envelopeDestinationPath string) error {
	var headers [][]string

	headers = append(headers, []string{HEADER_MESSAGE_ID, msg.ID})

	if msg.StreamID != "" {
		headers = append(headers, []string{HEADER_MESSAGE_STREAM, msg.StreamID})
	}

	contentHeaders := msg.EmbedContentHeaders()
	if msg.IsBroadcast {
		/*
		 * Broadcast headers are only base64 encoded, not encrypted.
		 */
		headers = append(headers, []string{HEADER_MESSAGE_CONTENT_HEADERS,
			"algorithm=none; value=" + base64.StdEncoding.EncodeToString(contentHeaders)})
	} else {
		/*
		 * Private messages define who may read the message, through hybrid encryption.
		 * A random "key" is used to seal the content through symmetric encryption. The
		 * key is then encrypted for each reader using respective public key and newly
		 * generated public key which is embedded.
		 *
		 * The default seal for the keys is NaCl Box, XChaCha20-Poly1305.
		 */
		headers = append(headers, []string{HEADER_MESSAGE_ENCRYPTION, msg.PayloadCipher.ToHeader()})

		accessList, err := msg.SealedAccessList()
		if err != nil {
			return err
		}
		headers = append(headers, []string{HEADER_MESSAGE_ACCESS, accessList})

		encryptedHeaders, err := crypto.Xchacha20Poly1305Encrypt(contentHeaders, msg.AccessKey)
		if err != nil {
			return err
		}
		headers = append(headers, []string{HEADER_MESSAGE_CONTENT_HEADERS,
			"algorithm=" + crypto.SYMMETRIC_CIPHER + "; " + "value=" + base64.StdEncoding.EncodeToString(encryptedHeaders)})
	}

	var envelopeDumpHeadersList []string // The list will be dumped as envelope
	var envelopeHeadersOrderList []string
	var envelopeChecksumValuesList []string
	for _, hPair := range headers {
		envelopeDumpHeadersList = append(envelopeDumpHeadersList, HeaderLine(hPair[0], hPair[1]))
		envelopeHeadersOrderList = append(envelopeHeadersOrderList, hPair[0])
		envelopeChecksumValuesList = append(envelopeChecksumValuesList, hPair[1])
	}

	envelopeChecksum := []byte(strings.Join(envelopeChecksumValuesList, ""))
	envelopeHexCheckSum, envelopeSumBytes := crypto.Checksum(envelopeChecksum)
	envelopeSignature := crypto.SignData(msg.Author.PublicSigningKey, msg.Author.PrivateSigningKey, envelopeSumBytes)

	// The last two are the checksum and its signature, and not part of the sum calculation
	envelopeDumpHeadersList = append(envelopeDumpHeadersList, HeaderLine(HEADER_MESSAGE_ENVELOPE_CHECKSUM, HeadersChecksumHeader(envelopeHexCheckSum, strings.Join(envelopeHeadersOrderList, ":"))))
	envelopeDumpHeadersList = append(envelopeDumpHeadersList, HeaderLine(HEADER_MESSAGE_ENVELOPE_SIGNATURE, SignatureHeader(envelopeSignature)))
	envelopeDumpStr := []byte(strings.Join(envelopeDumpHeadersList, "\n"))

	err := ioutil.WriteFile(envelopeDestinationPath, append(envelopeDumpStr, '\n'), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (msg *Message) Seal() (string, error) {
	storePath, err := storage.CreateLocalMessageDir(msg.Author.Address, msg.Content.SubjectID, msg.ID)
	if err != nil {
		return "", err
	}
	destBodyPath := filepath.Join(storePath, consts.MESSAGE_DIR_PAYLOAD_FILE_NAME)
	err = msg.SealBody(destBodyPath)
	if err != nil {
		return storePath, err
	}
	destEnvelopePath := filepath.Join(storePath, consts.MESSAGE_DIR_ENVELOPE_FILE_NAME)
	err = msg.SealEnvelope(destEnvelopePath)
	if err != nil {
		return storePath, err
	}
	return storePath, nil
}
